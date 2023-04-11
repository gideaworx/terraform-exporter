package registry

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/blang/semver/v4"
	"github.com/gideaworx/terraform-exporter-plugin-registry/registry"
	"github.com/gideaworx/terraform-exporter/runner"
	"github.com/olekukonko/tablewriter"
)

type ListAvailablePlugins struct {
	Name                 string `short:"n" default:"default" help:"The registry to list available plugins from"`
	ExcludeInstalled     bool   `short:"x" help:"If true, hide plugins that are already installed"`
	ShowAllArchitectures bool   `short:"a" help:"If true, show plugins available for any architecture, not just the local architecture"`
	r                    *PluginRegistries
}

func (l *ListAvailablePlugins) BeforeApply() error {
	var err error
	l.r, err = LoadFromDisk()
	return err
}

func (l *ListAvailablePlugins) Run(ctx *kong.Context) error {
	registry := l.r.Get(l.Name)
	if registry == nil {
		return fmt.Errorf("plugin registry %q not found", l.Name)
	}

	if err := registry.LazyLoad(); err != nil {
		return err
	}

	for i := range registry.Plugins {
		sort.Slice(registry.Plugins[i].Versions, func(a, b int) bool {
			sa, _ := semver.ParseTolerant(registry.Plugins[i].Versions[a].Version)
			sb, _ := semver.ParseTolerant(registry.Plugins[i].Versions[b].Version)
			return sa.GT(sb)
		})
	}

	sort.Slice(registry.Plugins, func(i, j int) bool {
		return registry.Plugins[i].Name < registry.Plugins[j].Name
	})

	headers := []string{"Name", "Description", "Latest Version"}

	columnColors := []tablewriter.Colors{
		{tablewriter.FgCyanColor},
		{tablewriter.FgWhiteColor},
		{tablewriter.FgGreenColor},
	}

	if !l.ExcludeInstalled {
		headers = append(headers, "Installed")
		columnColors = append(columnColors, tablewriter.Colors{tablewriter.FgWhiteColor})
	}

	headerColors := make([]tablewriter.Colors, len(headers))
	for i := 0; i < len(headers); i++ {
		headerColors[i] = tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}
	}

	table := tablewriter.NewWriter(ctx.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader(headers)
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeaderColor(headerColors...)
	table.SetColumnColor(columnColors...)

	for _, plugin := range registry.Plugins {
		if !l.exclude(plugin) {
			row := []string{plugin.Name, plugin.Description, plugin.Versions[0].Version}
			if !l.ExcludeInstalled {
				installed := ""
				if l.isInstalled(plugin) {
					installed = "✅"
				}

				row = append(row, installed)
			}
			table.Append(row)
		}
	}

	table.Render()
	return nil
}

func (l *ListAvailablePlugins) exclude(plugin registry.Plugin) bool {
	if l.ShowAllArchitectures && !l.ExcludeInstalled {
		return true
	}

	latestVersion := plugin.Versions[0]
	if !l.ShowAllArchitectures {
		foundArch := false
		goos := runtime.GOOS
		goarch := runtime.GOARCH

		arch := registry.TargetArchitecture(fmt.Sprintf("%s/%s", strings.ToLower(goos), strings.ToLower(goarch)))
		for availableArch := range latestVersion.DownloadInfo {
			if arch == availableArch || availableArch == registry.MultiArch {
				foundArch = true
				break
			}
		}

		if !foundArch {
			return false
		}
	}

	return l.ExcludeInstalled && l.isInstalled(plugin)
}

func (l *ListAvailablePlugins) isInstalled(plugin registry.Plugin) bool {
	boms, err := runner.LoadInstalledBOMs()
	if err != nil {
		log.Println(err)
		return false
	}

	for _, bom := range boms {
		if bom.Name == plugin.Name &&
			bom.Version.String() == plugin.Versions[0].Version {
			return true
		}
	}

	return false
}
