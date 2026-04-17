# Website Logo Design Document

**Date:** 2026-04-17
**Status:** Draft
**Author:** Codex

## Overview

Add a project-aligned website logo for Clash Subscription Manager and integrate it into the existing homepage hero without changing the product's current information hierarchy.

The logo should match the current dark, infrastructure-oriented UI: deep navy backgrounds, cyan-blue structural lines, and green accent highlights. The resulting asset set must support both full-size branding in the hero area and compact icon usage in smaller UI contexts.

## Requirements

### Functional Requirements

1. **Primary Logo Asset**: Provide a horizontal SVG logo made of a left-side icon plus the full wordmark `Clash Subscription Manager`.
2. **Compact Icon Asset**: Provide a standalone SVG icon derived from the same visual system for small-size use cases such as favicon or compact header treatments.
3. **Homepage Integration**: Display the primary logo in the homepage hero area.
4. **Preserve Existing Hierarchy**: Keep the existing page title `Clash 订阅管理` and summary copy; the logo acts as a brand header, not as a replacement for page title content.
5. **Dark Theme Compatibility**: Ensure the logo is legible on the current dark gradient hero background.

### Non-Functional Requirements

1. **Scalable Rendering**: Assets must remain sharp at multiple sizes, so SVG is the source format.
2. **Visual Consistency**: The logo must reuse the site's existing color language instead of introducing a new palette.
3. **Small-Size Clarity**: The compact icon must remain identifiable at reduced sizes.
4. **Low Layout Risk**: Homepage integration must not make the hero section feel crowded on desktop or mobile.
5. **Self-Contained Assets**: SVG assets should not rely on external CSS to remain legible and should avoid fragile runtime dependencies.

## Design Direction

### Brand Intent

The logo should communicate a professional infrastructure tool rather than a consumer-facing lifestyle product. The tone should feel stable, precise, and operational.

### Chosen Visual Motif

Use a `hub-and-nodes network` concept:

- A central node represents the management hub.
- Three outward connections represent multiple subscription sources or managed configuration paths.
- Smaller outer nodes show aggregation into one managed system.

This motif is the closest match to the product's actual behavior: collecting multiple subscriptions, transforming them, and managing them through one interface.

### Rejected Alternatives

- **Route-path symbol**: has stronger motion, but reads more like generic networking than subscription management.
- **Shield or badge frame**: feels too security-product-specific and would pull the identity away from the current product emphasis.

## Logo Composition

### Primary Logo

The primary logo uses a horizontal lockup:

- **Left**: geometric icon
- **Right**: single-line wordmark `Clash Subscription Manager`

This layout fits the current hero section width and supports immediate recognition in the homepage header.

### Icon Shape

The icon should use:

- A rounded outer silhouette between a circle and a softened hexagon
- One central node
- Three main connecting lines
- Three outer nodes with measured spacing

The geometry should feel ordered and technical, but not overly intricate. Avoid dense linework that would blur at smaller sizes.

### Wordmark

The wordmark should:

- Use the complete English project name
- Appear on one line
- Use a medium to semi-bold weight
- Avoid highly stylized futuristic forms

The wordmark should feel like a dependable software tool label, not a startup-style brand statement.

## Color and Styling

### Palette

The logo should derive from the existing UI palette:

- **Structural stroke / outline**: cyan-blue family already used in the interface
- **Accent nodes / highlights**: green accent already used in buttons and glow states
- **Optional neutral text tone**: white or near-white aligned with current text color

### Surface Behavior

- The logo should be designed for the existing dark hero panel first.
- It should not rely on large glow effects for recognition.
- Any highlight treatment should stay subtle and keep the mark readable in static SVG form.

## Homepage Integration

### Placement

Insert the primary logo as a dedicated brand row at the top of the existing `.hero-copy` block, before the current eyebrow label. The brand mark should introduce the product before the main Chinese page title, while keeping the title as the main content headline.

### Information Hierarchy

The intended hierarchy is:

1. Logo / product brand
2. Main page title: `Clash 订阅管理`
3. Supporting summary and action buttons

This keeps branding visible without reducing clarity for first-time users.

### Responsive Behavior

On smaller screens:

- The horizontal logo may reduce in width but should stay readable.
- If needed, the wordmark and icon may scale down together before any layout wrapping occurs.
- The hero layout should remain balanced and should not push metrics cards into awkward spacing.

## Implementation Scope

### Files Likely to Change

- `templates/index.html`
- `static/css/style.css`
- New asset files under `static/img/`

### Asset Deliverables

1. `static/img/logo-primary.svg`
2. `static/img/logo-icon.svg`

### Out of Scope

- Full rebrand of the application
- Replacing all existing titles with logo-only presentation
- Marketing artwork, social preview images, or splash screens
- Expanded icon system beyond the two core logo assets

## Accessibility and Usability

1. **Decorative vs informative use**: If inserted as an image element, the primary logo should have alt text that identifies the product.
2. **Contrast**: The mark must maintain enough contrast against the current hero background.
3. **No information loss**: Users should still understand the page even if the image fails to load because the textual title remains present.

## Testing Strategy

### Manual Verification

1. Confirm the primary logo renders correctly on desktop homepage layout.
2. Confirm the primary logo remains legible on mobile-width layout.
3. Confirm the compact icon remains identifiable at small dimensions.
4. Confirm the hero section spacing still feels intentional after integration.
5. Confirm the dark background provides enough contrast for the icon and wordmark.

### Regression Checks

1. Ensure hero buttons and metrics cards remain aligned after the new logo is added.
2. Ensure no existing content is removed or visually hidden by the added brand block.
3. Ensure static asset paths resolve correctly in the running app.

## Acceptance Criteria

The design is ready for implementation when:

1. The repository contains a primary horizontal SVG logo and a compact icon SVG.
2. The homepage hero displays the primary logo without replacing the existing page title.
3. The logo visually matches the current dark cyan/green design language.
4. Desktop and mobile layouts remain usable and visually balanced.
5. The compact icon is suitable for future favicon or narrow-space usage.
