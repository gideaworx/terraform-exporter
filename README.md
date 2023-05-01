# Terraform Exporter

![Latest Release](https://img.shields.io/github/v/release/gideaworx/terraform-exporter) ![MIT License](https://img.shields.io/github/license/gideaworx/terraform-exporter)

The `terraform-exporter` CLI is a pluggable tool that supports exporting resources
you have already created into [Terraform HCL][1] files. Plugins determine what 
resources should be exported and how to export them, as well as a terraform 
provider file. This tool generates a shell script to `terraform import` those 
resources to be used later.

By default, the tool does not know how to export anything, and leaves that up to 
the plugins to do. Plugins are hosted by registries, and a [default registry][2]
is configured in the tool. 

## Installing

### Homebrew (MacOS and Linux)

```bash
$ brew install gideaworx/tap/terraform-exporter
```

### Prebuilt Binaries

Visit the [releases][3] page.

### From source

```bash
$ go install github.com/gideaworx/terraform-exporter@latest
```

## Using

```
Usage: terraform-exporter <command>

Flags:
  -h, --help       Show context-sensitive help.
  -v, --version    Show the version and quit

Commands:
  export <command-name> <command-args> ...
    Export data to terraform files

  install-plugin (install,i) <plugin-name>
    Install a plugin

  remove-plugin (remove,rm) <plugin-name>
    Uninstall a plugin

  update-plugin (update,up) <plugin-name>
    Update a plugin

  help (h) <command-name>
    Show help for a plugin's exporter command

  list-plugins (ls)
    List installed plugins

  list-commands (lc)
    List commands provided by installed plugins

  registry show-catalog (ls)
    List all installed plugin registries

  registry available-plugins
    List all plugins available in a registry

  registry add
    Add a registry from which plugins can be installed

  registry remove (rm) <name>
    Remove a registry from the local catalog

Run "terraform-exporter <command> --help" for more information on a command.
```

## Developing a plugin

Follow the guides in the [plugin repository][4]

## License

`terraform-exporter` is released under the [MIT](./LICENSE) license.

## Contributing

Pull requests are welcome! All contributors are bound by the [Code of Conduct][5].
Before opening a pull request, please open an [issue][6] so it can be triaged.
Please ensure, to the best of your ability, that the issue actually lies within the
CLI itself and not a plugin before opening an issue.

<!-- Links -->
[1]: https://terraform.io
[2]: https://plugin-registry.gideaworx.io
[3]: ../../releases
[4]: https://github.com/gideaworx/terraform-exporter-plugin
[5]: CODE_OF_CONDUCT.md
[6]: ../../issues/new