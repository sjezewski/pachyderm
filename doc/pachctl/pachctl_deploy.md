## ./pachctl deploy

Print a kubernetes manifest for a Pachyderm cluster.

### Synopsis


Print a kubernetes manifest for a Pachyderm cluster.

```
./pachctl deploy [amazon bucket id secret token region volume-name volume-size-in-GB | google bucket volume-name volume-size-in-GB | microsoft container storage-account-name storage-account-key volume-uri volume-size-in-GB]
```

### Options

```
  -d, --dev                           Don't use a specific version of pachyderm/pachd.
      --dry-run                       Don't actually deploy pachyderm to Kubernetes, instead just print the manifest.
  -p, --host-path string              the path on the host machine where data will be stored; this is only relevant if you are running pachyderm locally. (default "/tmp/pach")
  -r, --registry                      Deploy a docker registry along side pachyderm. (default true)
      --rethinkdb-cache-size string   Size of in-memory cache to use for Pachyderm's RethinkDB instance, e.g. "2G". Default is "768M". Size is specified in bytes, with allowed SI suffixes (M, K, G, Mi, Ki, Gi, etc) (default "768M")
  -s, --shards int                    The static number of shards for pfs. (default 32)
```

### Options inherited from parent commands

```
  -v, --verbose   Output verbose logs
```

### SEE ALSO
* [./pachctl](./pachctl.md)	 - 

###### Auto generated by spf13/cobra on 31-Oct-2016
