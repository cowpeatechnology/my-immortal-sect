# Sect Map Building Asset Footprint And State Spec

**Status**: draft  
**Owner**: `art_asset_producer`  
**Scope**: `F-004` sect-map building assets only  
**Last Updated**: 2026-04-22

## Purpose

Freeze the first durable building-art contract for the portrait sect map so imported building assets share one rule for:

- rectangular tile occupancy
- footprint-to-render-size mapping
- bottom anchor behavior on the isometric grid
- state-driven readability treatment

This spec covers importable sect-map building assets only. It does not change runtime code, combat rules, or session logic.

## Authority Inputs

- `AGENTS.md`
- `docs/features/F-004-sect-map-m1-gameplay-foundation.md`
- `docs/vision/design-decisions.md`
- Current sect-map grid contract from `client/my-immortal-sect/assets/resources/tilemaps/sect-map/sect-map-demo.tmx`
- Current runtime building definitions from `client/my-immortal-sect/assets/scripts/app/sect-map-bootstrap.ts`
- Current raster measurements from `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-raster/*.png`

## Scope Boundary

This spec defines the asset contract for the five current building classes:

- `main_hall`
- `disciple_quarters`
- `warehouse`
- `herb_garden`
- `guard_tower`

This spec does not introduce:

- non-rectangular footprints
- a second damaged paint pass for every building
- per-state full repaint variants
- platform-container requirements

## Grid Contract

The current sect map uses:

- map orientation: isometric
- map size: `14 x 14`
- tile size: `128 x 64`

Rectangular occupancy is mandatory.

- Each building occupies one axis-aligned rectangle in tile space.
- `origin` is the minimum `col,row` corner of that rectangle.
- Occupied tiles are `origin.col .. origin.col + width - 1` and `origin.row .. origin.row + height - 1`.
- Decorative silhouette may rise above the footprint, but it must not claim extra occupied tiles outside the rectangle.

## Render-Size Rule

### 1. Occupancy frame

For a building footprint `w x h` on the current `128 x 64` isometric grid:

- occupancy frame width = `(w + h) * 64`
- occupancy frame height = `(w + h) * 32`

This is the diamond-frame area that the runtime footprint already reserves visually.

### 2. Sprite anchor

All building sprites use one anchor rule:

- anchor point: bottom-center of the sprite
- placement point: bottom-center of the occupancy frame
- added visual height must grow upward, not downward

Practical result:

- no extra bottom padding below the footprint base
- no lateral overhang beyond the occupancy frame unless a later spec explicitly approves it

### 3. Baseline raster canvas

The first durable raster target keeps width equal to the occupancy frame width and uses class-specific vertical headroom above that frame.

Formula:

- raster width = occupancy frame width
- raster height = occupancy frame height + headroom

## Canonical Building Class Table

The current runtime art pack already matches the following baseline raster sizes, so this spec freezes them as the first durable targets.

| Runtime Key | Footprint Tiles | Occupancy Frame px | Headroom px | Baseline Raster px | Visual Intent |
| --- | --- | --- | --- | --- | --- |
| `main_hall` | `3 x 3` | `384 x 192` | `144` | `384 x 336` | Primary sect anchor, broad roof mass, highest silhouette among economy buildings |
| `disciple_quarters` | `2 x 3` | `320 x 160` | `112` | `320 x 272` | Vertical residential block, readable dorm silhouette without tower height |
| `warehouse` | `2 x 2` | `256 x 128` | `96` | `256 x 224` | Dense storage mass, stable low-mid profile |
| `herb_garden` | `2 x 2` | `256 x 128` | `64` | `256 x 192` | Lowest farming silhouette, open-air growing beds and light roof detail |
| `guard_tower` | `1 x 2` | `192 x 96` | `192` | `192 x 288` | Tallest defensive vertical, narrow footprint with strong top read |

## Variant Policy

Per building class, V1 keeps one primary class illustration:

- one canonical building base art in `active` state

State readability is then added through shared treatment layers instead of full repaints.

Do not multiply variants into:

- separate blueprint painting per building
- separate supplied painting per building
- separate constructing painting per building
- separate damaged painting per building
- separate level-2 repaint for every building

## State Treatment Plan

### Blueprint

Use the base footprint and silhouette language, but do not require unique blueprint art.

Required treatment:

- footprint plate remains visible
- shared `planned` signal is the primary remote read
- class silhouette may be shown as a low-opacity ghost only if the runtime needs it

### Active

This is the canonical exported building art.

Required treatment:

- full-color class illustration
- silhouette must read clearly at sect-map zoom without labels
- building identity comes from shape first, color second

### Damaged

Do not draw a second full damaged asset for each building in V1.

Required treatment:

- reuse active base art
- add shared damaged readability through runtime tint / crack / disabled signal treatment
- preserve the class silhouette so players still recognize what was hit

Two damage reads must remain visible:

- damaged but functioning
- disabled / collapsed enough to require repair

### Upgrade / Level Readability

Do not create full level-2 repaint sets for every building.

Required treatment:

- shared upgrade/supplied/constructing signals remain the first readability surface
- class-specific top add-ons are optional only when the upgraded state must read from far view
- the first priority is `guard_tower`, because its level-2 readiness matters directly to the raid loop

Default rule:

- if a level change can read through silhouette topper or signal, do that
- if it needs a full repaint, defer until gameplay proves the cheaper read is insufficient

## Building Readability Notes

### `main_hall`

- Must read as the sect’s emotional and spatial center.
- Broad roof mass and warm palette should dominate nearby economy buildings.

### `disciple_quarters`

- Must read as habitation, not shrine or storage.
- Favor stacked room rhythm, hanging cloth, or warm-lit window grouping over heavy fortification cues.

### `warehouse`

- Must read as stockpile and logistics.
- Favor dense massing, banded doors, cargo rhythm, and compact roof language.

### `herb_garden`

- Must read as cultivation / planting at map scale.
- Keep the silhouette low and open so it does not compete with civic buildings.

### `guard_tower`

- Must read as defense from the farthest gameplay camera.
- Height and crown shape matter more than facade detail.
- This is the one building class where upgrade readability is most likely to justify a lightweight topper variant later.

## Import And Naming Contract

The current asset roots remain valid:

- canonical source: `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-svg/`
- importable raster: `client/my-immortal-sect/assets/resources/generated-buildings/sect-map-raster/`

Building basename rule:

- SVG source: `<runtime_key>.svg`
- raster export: `<runtime_key>.png`

Examples:

- `main_hall.svg -> main_hall.png`
- `guard_tower.svg -> guard_tower.png`

Shared state signal assets remain separate from building-class art:

- `building_signal_planned`
- `building_signal_supplied`
- `building_signal_constructing`
- `building_signal_damaged`
- `building_signal_disabled`

## Current Alignment Note

The live client building definitions and the shared config do not yet fully agree on every footprint.

This spec freezes the building-art footprint contract to the current sect-map runtime building pack and grid math:

- it matches the current raster outputs already imported by the client
- it preserves the rectangular-only occupancy rule
- it gives engineering one durable target to reconcile into shared config next

Until that reconciliation lands, new building art should follow this document rather than inventing a third footprint table.

## Done-When Mapping

This spec satisfies the current subfunction when used as the building-art source of truth for:

- rectangular footprint occupancy
- baseline raster target size
- sprite anchor behavior
- blueprint / active / damaged / upgrade readability policy
