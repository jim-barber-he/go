
# ssm

A tool for manipulating parameters in the AWS SSM Parameter Store.

The tool is somewhat tailored to the environment at my workplace.

Each of the sub-commands accepts an environment name as the first argument.
This is one of `dev`, `test*`, or `prod*`. The command maps these to the `hetest`, `hetest`, or `heaws` AWS profile respectively.

The environments also influence where the SSM parameters are looked for if not fully qualified by starting with a slash (/).
Non-qualified parameters will be prefixed with `/helm/minikube/`, `/helm/test*/`, or `/helm/prod*/`.
The `minikube` in the path is a legacy path for the development environments at my work place.
The `/helm/` prefix for all of them is a strange naming convention where the name of the product using these parameters was used
for the initial path.

By default it uses a KMS key with the alias of `parameter_store_key` for storing SecureString values.

## Usage

### ssm

```
Usage:
  ssm [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  delete      Delete a parameter from the SSM parameter store
  get         Retrieve a parameter from the AWS SSM parameter store
  help        Help about any command
  list        List parameters from the SSM parameter store below a supplied path
  put         Store a parameter and its value in the AWS SSM parameter store

Flags:
  -h, --help             help for ssm
      --profile string   AWS profile to use
      --region string    AWS region to use (default "ap-southeast-2")

Use "ssm [command] --help" for more information about a command.
```

### ssm completion

Use for setting up command line completion for a shell.
e.g.
```shell
source <(ssm completion zsh)
```

```
Usage:
  ssm completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion

Global Flags:
      --profile string   AWS profile to use
      --region string    AWS region to use (default "ap-southeast-2")
```

### ssm delete

Delete a parameter from the SSM parameter store.

```
Usage:
  ssm delete [flags] ENVIRONMENT PARAMEMETER

Flags:
  -h, --help   help for delete

Global Flags:
      --profile string   AWS profile to use
      --region string    AWS region to use (default "ap-southeast-2")
```

### ssm get

Retrieve a parameter from the AWS SSM parameter store.

```
Usage:
  ssm get [flags] ENVIRONMENT PARAMETER

Flags:
  -f, --full   Show all details for the parameter
  -h, --help   help for get

Global Flags:
      --profile string   AWS profile to use
      --region string    AWS region to use (default "ap-southeast-2")
```

### ssm list

List variables from the SSM parameter store below the supplied path.

```
Usage:
  ssm list [flags] ENVIRONMENT [PATH]

Flags:
  -b, --brief          Show parameter = value output
  -f, --full           Show additional details for each parameter
  -h, --help           help for list
  -r, --recursive      Recursively list parameters below the parameter store path
  -s, --safe-decrypt   Slower decrypt that can handle errors

Global Flags:
      --profile string   AWS profile to use
      --region string    AWS region to use (default "ap-southeast-2")
```

### ssm put

Store a parameter and its value in the AWS SSM parameter store.

```
Usage:
  ssm put [flags] ENVIRONMENT PARAMETER VALUE
  ssm put [flags] ENVIRONMENT PARAMETER --file FILE

Flags:
  -f, --file string     Get the value from the file contents
  -h, --help            help for put
      --key-id string   The ID of the KMS key to encrypt SecureStrings (default "alias/parameter_store_key")
      --secure          Store the value as a SecureString
  -v, --verbose         Show the value set for the parameter

Global Flags:
      --profile string   AWS profile to use
      --region string    AWS region to use (default "ap-southeast-2")
```
