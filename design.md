# Design System Strategy: The Silent Precisionist

## 1. Overview & Creative North Star
The Creative North Star for this system is **"The Invisible Assistant."**

Unlike typical utilities that demand attention through heavy borders and aggressive alerts, this system is rooted in the concept of *functional transparency*. It is designed to feel like a high-end, native extension of a professional workstation—oscillating between "quiet background state" and "surgical precision" when active.

We break the "template" look by rejecting the standard grid in favor of **Intentional Asymmetry**. Key information is anchored to a rigid baseline, but action elements are allowed to float within generous whitespace. By using high-contrast typography scales—pairing large, airy `display` type with tiny, ultra-legible `label` caps—we create an editorial rhythm that feels premium and bespoke.

---

## 2. Colors & Surface Philosophy
The palette is a sophisticated study in neutrals, punctuated by a single, focused accent (`primary: #446273`).

### The "No-Line" Rule
Traditional 1px solid borders are strictly prohibited for sectioning. Boundaries must be defined through **Background Color Shifts**.
* **Implementation:** A sidebar using `surface-container-low` should sit against a main content area of `surface`. The "edge" is created by the eye perceiving the tonal shift, not a drawn line.

### Surface Hierarchy & Nesting
Think of the UI as a series of stacked, fine-milled paper sheets.
* **Base:** `surface` (#f8f9fa)
* **Structural Sections:** `surface-container-low` (#f1f4f6)
* **Active Modules/Cards:** `surface-container-lowest` (#ffffff) to create a subtle "pop" of cleanliness.
* **Deep Contextual Areas:** `surface-container-high` (#e3e9ec) for utility bars or footers.

### The "Glass & Gradient" Rule
To elevate the utility from "basic" to "premium," floating panels (like renaming previews) should utilize **Glassmorphism**.
* **Value:** `surface_variant` at 70% opacity with a `20px` backdrop-blur.
* **Signature Texture:** Main action buttons should not be flat. Use a subtle linear gradient from `primary` (#446273) to `primary_dim` (#385666) at a 145-degree angle to give the element "mass" and tactile quality.

---

## 3. Typography
We utilize **Inter** as the sole typeface, relying on extreme scale and weight shifts to communicate hierarchy.

* **Display & Headlines:** Use `display-sm` for empty states or "Zero Tasks" screens. It should feel like a magazine header—understated but authoritative.
* **Title & Body:** `title-sm` is your workhorse for file names. Use `body-md` for metadata.
* **The Precision Label:** `label-sm` should be used for file extensions and status indicators. When using `label-sm`, apply a `0.05rem` letter-spacing to enhance the "technical" feel.

The hierarchy moves from "Human Editorial" (Large Display) to "Machine Precision" (Small Labels), mirroring the app’s role as a bridge between the user and the file system.

---

## 4. Elevation & Depth
In this system, depth is a whisper, not a shout.

### Tonal Layering
Avoid shadows for static elements. Instead, stack `surface-container-lowest` on top of `surface-container-low`. The `0.125rem` difference in tonal value is sufficient to denote hierarchy.

### Ambient Shadows
For "Floating" elements (e.g., a file-drop overlay), use an **Ambient Shadow**:
* **Blur:** `24px`
* **Spread:** `-4px`
* **Color:** `on-surface` (#2b3437) at **6% opacity**.
This mimics natural light diffracted through a window, rather than a digital drop shadow.

### The "Ghost Border"
If a container requires a boundary (e.g., a text input), use the **Ghost Border**:
* **Token:** `outline-variant` (#abb3b7) at **15% opacity**.
* **Constraint:** Never use 100% opacity for borders; it breaks the "featherweight" illusion.

---

## 5. Components

### Buttons
* **Primary:** Gradient of `primary` to `primary_dim`. Shape: `md` (0.375rem). Text: `label-md` in `on-primary`.
* **Secondary:** Ghost style. No background, `Ghost Border` on hover, `primary` text.
* **Tertiary:** Flat `surface-container-high` background, `on-surface` text. Used for low-priority utility tasks.

### Precision Renaming Inputs
* **Style:** No bottom line. Use a `surface-container-lowest` fill with a `Ghost Border`.
* **Focus State:** The border transitions to `primary` at 40% opacity with a subtle `2px` outer glow of `primary_fixed`.

### Files & Lists
* **The Divider Rule:** Strictly forbid horizontal lines between files.
* **Alternative:** Use `spacing: 1` (0.35rem) as a vertical gap and alternate background colors between `surface` and `surface-container-low` only if the list exceeds 20 items. Otherwise, let whitespace define the row.

### Status Chips
* **Active Renaming:** `tertiary_container` background with `on-tertiary_container` text. Shape: `full`.
* **Success:** `primary_container` with `on-primary_container`.

---

## 6. Do’s and Don'ts

### Do:
* **Use Asymmetry:** Align the app title to the far left and the primary action to the far right with nothing in between to create "tension" and "air."
* **Respect the Spacing Scale:** Use `spacing: 8` (2.75rem) for outer margins to ensure the app feels like it’s floating on the desktop.
* **Embrace the Blur:** Use backdrop blurs on any element that overlaps another to maintain a sense of "lightness."

### Don't:
* **Don't use pure Black:** Use `on-background` (#2b3437) for text. Pure black is too heavy for a "featherweight" utility.
* **Don't use Standard Shadows:** Avoid the default CSS `box-shadow: 0 2px 4px rgba(0,0,0,0.5)`. It is too "web-standard" for this editorial experience.
* **Don't Over-Round:** Stick to the `md` (0.375rem) scale. Rounding that is too aggressive (like `xl`) feels "toy-like" rather than "precision-tooled."