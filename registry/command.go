package registry

type Command struct {
	ShowCatalog      *ListInstalledRegistries `cmd:"" aliases:"ls" help:"List all installed plugin registries"`
	AvailablePlugins *ListAvailablePlugins    `cmd:"" help:"List all plugins available in a registry"`
	Add              *AddRegistryCommand      `cmd:"" help:"Add a registry from which plugins can be installed"`
	Remove           *RemoveRegistryCommand   `cmd:"" aliases:"rm" help:"Remove a registry from the local catalog"`
}
