# Friction log

Raw notes from dogfooding the MVP. One entry per friction, however
small — a miss, a surprise, a wish. No polish; volume beats quality
here. v2 gets scoped from this file.

Entry format (copy the block):

```
## <date> — <site or action>
- expected:
- got:
- hurt (1=shrug, 3=annoying, 5=rage-quit):
- idea (optional):
```

---

## 2026-07-18 — example (delete me)

- expected: `manul docs.astro.build` to show their docs
- got: "No markdown here" — their llms.txt 404s despite being listed in a directory
- hurt: 2
- idea: start-page directory entries should be re-verified periodically

## 2026-07-18 — directory.llmstxt.cloud as landing page

- expected: the llms.txt aggregator directories to be browsable in manul
- got: directory.llmstxt.cloud publishes only a 4-line stub llms.txt — its
  actual site list is HTML-only; llms-txt.io and llmsdirectory.com are
  similar stubs. Only llmstxthub.com serves a real markdown directory
  (649 KB, auto-generated). Worked around with the new `home` config key.
- hurt: 3
- idea: strong evidence for the annuaire brick — the markdown web has
  almost no markdown-native directory; manul could generate/host its own
