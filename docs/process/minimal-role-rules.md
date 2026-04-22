# Minimal Role Rules

This project keeps role behavior intentionally small.

## Shared Rule

When notified:

1. read `docs/project/development.active.json`
2. if `owner_role` is not your role, stop
3. if `owner_role` is your role, read `must_read`
4. act only inside the current subfunction objective

## Worker Roles

Worker roles such as `engineer` or `art_asset_producer`:

- execute the scoped subfunction
- write `submit` through `node .coordex-v2/bin/coordex-event.mjs --actor <role>` when ready
- write `block` through the same helper if stuck
- after a supervisor `reject`, re-check the reject note and evidence before deciding whether a bounded fix is actually needed

Worker roles do not:

- choose the next role
- accept their own work
- rewrite the full plan

## Supervisor

Supervisor:

- reviews submitted work
- writes `accept` or `reject` through `node .coordex-v2/bin/coordex-event.mjs --actor supervisor`
- does not need to manually route a rejected task back; Coordex hands the same subfunction back to the original worker

Supervisor does not:

- perform ordinary worker-owned implementation
- rewrite broad plan text during normal review
- own normal plan subfunctions unless runtime code is explicitly extended for that case

## Coordex

Coordex:

- updates `development-plan.json`
- overwrites `development.active.json`
- appends `start` and other system-owned events
- advances to the next executable state

Coordex does not invent evidence or acceptance results.
