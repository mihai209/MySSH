# MySSH

MySSH este un client SSH UI orientat pe:

- lightweight, fara Electron
- profiluri SSH usor de gestionat
- fundatie sigura pentru parole, chei private si integrare cu `ssh-agent`

## Etapa curenta

Acest prim pas construieste fundatia aplicatiei:

- model de profil SSH
- validare stricta pentru campurile principale
- storage local pentru profiluri, fara a salva secretele in clar
- serviciu intern care va fi folosit de UI in etapa urmatoare

## Model de securitate pentru MVP

- Profilurile sunt persistate in JSON local.
- Parolele si cheile private nu sunt persistate in clar.
- Campul `secret_ref` este rezervat pentru integrarea ulterioara cu un secret store OS keyring sau alt mecanism dedicat.
- Tipul de autentificare `agent` este pregatit pentru integrare cu `ssh-agent`.

## Urmatorii pasi

1. UI nativ lightweight in Go
2. Secret store real pentru parole si chei
3. Conectare SSH prin clientul de sistem sau biblioteca dedicata
4. Terminal interactiv si management de sesiuni
