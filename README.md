Helm outdated dependencies
--------------------------

Helm plugin to list and update outdated dependencies of a Helm chart.

## Install

```
helm plugin install https://github.com/sapcc/helm-outdated-dependencies
```

## Usage

```
Helm plugin to manage outdated dependencies of a Helm chart.

Examples:
  $ helm outdated-dependencies list <pathToChart> 										- Checks if there's a newer version of any dependency available in the specified repository.
  $ helm outdated-dependencies list <pathToChart> --repositories repo1.corp,repo2.corp 	- Checks if there's a newer version of any dependency available only using the given repositories. 

  $ helm outdated-dependencies update <pathToChart> 							- Updates all outdated dependencies to the latest version found in the repository.
  $ helm outdated-dependencies update <pathToChart> --increment-chart-version	- Updates all outdated dependencies to the latest version found in the repository and increments the version of the Helm chart.

```

## RELEASE

Releases are done via [goreleaser](https://github.com/goreleaser/goreleaser).  
Tag the new release, export the `GORELEASER_GITHUB_TOKEN` (needs `repo` scope) and run `make release`.
