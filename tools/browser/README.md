# Browser Tools

This subtree holds the ChatGPT web, Chrome bridge, CDP observer, and browser-driven image generation utilities.

Current layout:

- `docs/`: operator guides, workflow notes, and troubleshooting references
- `scripts/`: Python and Node entrypoints used to run browser automation workflows
- `templates/`: reusable batch manifests and sample job specs
- `extension/`: Chrome extension source used by the browser bridge flow

Rules:

- Keep runnable entrypoints under `scripts/`.
- Keep reusable manifests under `templates/`.
- Keep operator-facing instructions under `docs/`.
- Write generated files and captures to `workspace/output/browser/`, not inside this subtree.
- Keep unrelated asset-pipeline work out of this directory.
