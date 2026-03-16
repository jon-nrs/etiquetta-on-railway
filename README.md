<p align="center">
  <img src="ui/public/logo-complete.png" alt="Etiquetta" width="120" >
</p>

<p align="center">
  <strong>Self-hosted web analytics. Privacy-focused. Single binary.</strong>
</p>

<p align="center">
  <a href="https://github.com/caioricciuti/etiquetta/actions/workflows/ci.yml">
    <img src="https://github.com/caioricciuti/etiquetta/actions/workflows/ci.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/caioricciuti/etiquetta/releases/latest">
    <img src="https://img.shields.io/github/v/release/caioricciuti/etiquetta" alt="Release">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/license-GPL--3.0-blue.svg" alt="License">
  </a>
</p>

---

## Etiquetta on Railway

This is a fork of [Etiquetta](https://etiquetta.com) with the addition of a nixpacks file to enable the service to be built on Railway. Some additional configuration in Railway is also necessary. 

To run Etiquetta on Railway:

1. Fork this repo
2. Connect your GitHub to Railway
3. In your Railway Dashboard, hit `+ New`
4. Choose `GitHub Repository`
5. Choose the `Railway config` branch of the repo you just created
6. Open the configuration panel, and [add the variables shown here](https://github.com/caioricciuti/etiquetta?tab=readme-ov-file#configuration)
7. Add another variable called `ALLOWED_HOSTS`. The value should be your public Railway URL (found in the Settings tab under Public Networking)
8. Add another variable called `RAILPACK_PACKAGES`, with a value of `bun@latest`
9. In Settings, add the following:
	- **Public networking** (optional): Add a custom domain (e.g. analytics.mydomain.com) to obfuscate your Railway app URL in your website source code 
	- **Builder**: Railpacks
	- **Custom Build Command**: make all
	- **Custom Start Command**: ./bin/etiquetta serve
10. Close the configuration panel, hit `+ New` and choose `Volume`, attach it to your Etiquetta service
11. Add `/app/data` as the mount path. This ensures your Etiquetta data is retained if your Railway services crashes or your redeploy it
10. Hit `Deploy`, and [continue your onboarding from here](https://github.com/caioricciuti/etiquetta?tab=readme-ov-file#tracking-setup).
