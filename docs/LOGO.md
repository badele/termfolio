# termfolio logo

## generate

```bash
nix shell github:badele/splitans nixpkgs#bit-logo nixpkgs#ansilove

bit -font larceny "TERM\nFOLIO" > logo.ans
splitans logo.ans -E cp437 -F ansi -S > logo-cp437.ans
icy_draw logo-cp437.ans
splitans -e cp437 -F neotex -S logo-cp437.ans > logo.neo
ansilove -S -o termfolio-logo.png logo-cp437.ans
```

## Source

- Tools
  - [ansilove](https://github.com/ansilove/ansilove)
  - [chafa](https://github.com/hpjansson/chafa)
  - [bit](https://github.com/superstarryeyes/bit)
  - [icy_draw](https://github.com/mkrueger/icy_tools)

```
```
