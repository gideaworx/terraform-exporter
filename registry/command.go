package registry

type RegistryCommand struct {
	ListInstalledRegistries *ListInstalledRegistries `cmd:"" help:"List all installed plugin registries"`
	ListAvailablePlugins    *ListAvailablePlugins    `cmd:"" help:"List all plugins available in a registry"`
	AddRegistry             *AddRegistryCommand      `cmd:"" help:"Add a registry from which plugins can be installed"`
}
