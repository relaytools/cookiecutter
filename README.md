# cookiecutter

Cookiecutter is a suite of tools for interacting with the relaytools API and managing a cluster of haproxy and strfry.

Part of the relaytools stack.

## Getting started

```bash
# setup the env with the nostr private key
cp env.develop .cookiecutter.env
go build
./cookiecutter --help
```

### Actions

Adding or removing a pubkey from a relay's allowlist
```
./cookiecutter action allowlist add --relay <relayID> --pubkey xyz --reason "team"
./cookiecutter action allowlist remove --relay <relayID> --pubkey xyz --reason "team"
```

### Deployment

This tool is also used in the deployment of a relaytools cluster.

* cookiecutter strfrydeploy (deploys strfry)
* cookiecutter haproxydeploy (deploys haproxy)
