## Logo genration

```bash
splitans CODEF_output.ans -W 109 -S > logo.neo
splitans -f neotex -E cp437 -F ansi logo.neo -S > logo.ans
ansilove -c 109 logo.ans
```

## Source

- Tool
  - [Ansilove](https://www.ansilove.org/)
  - [Splitans](https://github.com/badele/splitans)
- Font
  - BIGL2 from
    [CODEF](https://codef-ansi-logo-maker-api.santo.fr/api.php?text=term%20folio&font=91&spacing=1&spacesize=5&vary=0)
