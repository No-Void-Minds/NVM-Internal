---
name: NVM Internal System
colors:
  surface: '#fcf9f2'
  surface-dim: '#dcdad3'
  surface-bright: '#fcf9f2'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f6f3ec'
  surface-container: '#f1eee7'
  surface-container-high: '#ebe8e1'
  surface-container-highest: '#e5e2db'
  on-surface: '#1c1c18'
  on-surface-variant: '#434840'
  inverse-surface: '#31312c'
  inverse-on-surface: '#f3f0e9'
  outline: '#73796f'
  outline-variant: '#c3c8bd'
  surface-tint: '#4a6544'
  primary: '#284225'
  on-primary: '#ffffff'
  primary-container: '#3f5a3a'
  on-primary-container: '#b1d0a8'
  inverse-primary: '#b0cfa7'
  secondary: '#645d53'
  on-secondary: '#ffffff'
  secondary-container: '#ece1d4'
  on-secondary-container: '#6b6359'
  tertiary: '#423b30'
  on-tertiary: '#ffffff'
  tertiary-container: '#5a5246'
  on-tertiary-container: '#d1c6b6'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#ccebc1'
  primary-fixed-dim: '#b0cfa7'
  on-primary-fixed: '#072107'
  on-primary-fixed-variant: '#334d2e'
  secondary-fixed: '#ece1d4'
  secondary-fixed-dim: '#cfc5b9'
  on-secondary-fixed: '#201b13'
  on-secondary-fixed-variant: '#4c463c'
  tertiary-fixed: '#ece1d1'
  tertiary-fixed-dim: '#d0c5b5'
  on-tertiary-fixed: '#201b11'
  on-tertiary-fixed-variant: '#4d463a'
  background: '#fcf9f2'
  on-background: '#1c1c18'
  surface-variant: '#e5e2db'
typography:
  headline-lg:
    fontFamily: IBM Plex Sans
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.01em
  headline-md:
    fontFamily: IBM Plex Sans
    fontSize: 18px
    fontWeight: '500'
    lineHeight: 26px
    letterSpacing: 0em
  body-lg:
    fontFamily: IBM Plex Sans
    fontSize: 15px
    fontWeight: '400'
    lineHeight: 24px
    letterSpacing: 0em
  body-sm:
    fontFamily: IBM Plex Sans
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
    letterSpacing: 0em
  label-md:
    fontFamily: IBM Plex Sans
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: 0.01em
  caption:
    fontFamily: IBM Plex Sans
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
    letterSpacing: 0em
  specimen-code:
    fontFamily: IBM Plex Mono
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: 0.02em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 1.25rem
  gutter-mobile: 0.75rem
  margin: 2rem
  margin-mobile: 1rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 1rem
  space-lg: 1.5rem
  space-xl: 2.5rem
---

## Brand & Style

This design system establishes a clinical yet warm, utilitarian atmosphere tailored for an Indonesian apparel studio operating under a disciplined specimen-lab ethos. The aesthetic fuses archival garment documentation with strict operational efficiency: quiet, precise, purposeful, and free of decorative excess.

### Visual Aesthetic & Philosophy
- **Editorial Lab & Specimen Register:** Interfaces function like botanical or textile specimen archives. Visual hierarchy is established through meticulous typographic contrast and whitespace rather than decorative containers or nested panels.
- **Tone & Demeanor:** Calm, deliberate, clinical yet organic. It speaks with the quiet assurance of a material laboratory rather than a frantic SaaS dashboard.
- **Copy & Localization Protocol:** 
  - All interface navigation, controls, and system communications use concise Bahasa Indonesia (`"Buat task"`, `"Simpan perubahan"`, `"Masuk ke sistem"`).
  - Workflow status names remain strictly in English (`Requested`, `On Progress`, `Ready`, `Completed`) to maintain consistent operational nomenclature across factory, cutting room, and design teams.
  - Specimen and task titles use the sample prefix format `Contoh: [Nama Kegiatan]`.
  - Anonymized role actors use uniform laboratory placeholders (`Staf A`, `Staf B`, `Manajer A`).
- **Strict Prohibitions:** No glassmorphism, no backdrop blurs, no saturated blue/purple tech accents, no glowing outlines, no decorative dot matrices or blueprint grids, no emojis, sparkle motifs, or artificial stock assets.

## Colors

The palette takes inspiration from unbleached cotton, raw canvas, stone, and garment laboratory uniforms. Color functions strictly as semantic status indicators or primary focal anchors.

### Base Surfaces & Content
- **Canvas / Background:** `#F4F1EA` (Warm unbleached paper / linen).
- **Surface / Card / Table Row:** `#FFFFFF` (Pure clinical contrast against `#F4F1EA`).
- **Primary Text:** `#1F1D1A` (Deep carbon, minimum 11.5:1 contrast against white).
- **Secondary / Muted Text:** `#635C52` (Ground earth grey, compliant for secondary metadata).
- **Structural Outlines & Rules:** `#8F8678` (1px crisp boundary defining table borders, dividers, card frames, and form inputs).

### Brand & Interactive Accent
- **Primary Accent (`#3F5A3A`):** Deep vegetable olive. Used for primary execution buttons, selection indicators, active radio points, and high-priority filters. Paired exclusively with `#FFFFFF` text.
- **Focus Ring:** `2px solid #3F5A3A` with an offset of `2px solid #F4F1EA` or `#FFFFFF` depending on the current canvas.

### Status Tokens (Strict Two-Tone Muted Lab Chips)
Status tokens must always display readable, typed text labels—never standalone dots:
- **Requested:** Background `#E6E1D7`, Text `#4A443B`
- **On Progress:** Background `#E3E8DA`, Text `#3D4F2C`
- **Ready:** Background `#F1E3C8`, Text `#6B4A12`
- **Completed:** Background `#DDE5E3`, Text `#2F4A45`

### Urgency Indicators (Explicit Verbal Labels)
Urgency is never communicated via color alone. Every indicator must spell out the urgency word alongside the token:
- **Urgent:** `#9C3A22` (Terracotta red)
- **High:** `#8A5A12` (Ochre rust)
- **Medium & Low:** `#635C52` (Secondary neutral)

## Typography

The typographic system pairs the disciplined rationality of **IBM Plex Sans** with the technical rigor of **IBM Plex Mono**. Monospaced type is strictly reserved for technical specimen codes, task numbers, timestamps, and cut sheet identifiers.

### Rules & Hierarchy
- **No All-Caps Headings:** Headings must use standard sentence casing. Avoid high-tracking capital treatments to maintain an understated, functional editorial tone.
- **Normal Tracking:** Avoid artificial letter expansion (`tracking-widest`). Readability must be direct and natural.
- **IBM Plex Mono Enforcement:** Reserved specifically for IDs (`[T-0042]`, `LOT-881`, `2024-W12-01`). Do not use monospace for navigation links, status words, or form labels.
- **Typographic Scale:**
  - `headline-lg`: 24px, font-semibold (Page titles, view headers)
  - `headline-md`: 18px, font-medium (Section titles, sheet headings)
  - `body-lg`: 15px, font-normal (Primary content text, form inputs, table data)
  - `caption` & `label-md`: 13px, font-normal / font-medium (Metadata, helper copy, action labels)
  - `specimen-code`: 13px, font-medium monospace (Specimen and batch identifiers)

## Layout & Spacing

Whitespace serves as the structural material of the UI. Rather than nesting gray containers inside white boxes inside tinted frames, sections are delineated by whitespace gaps, hairline dividers (`#8F8678`), and column alignment.

### Grid & Page Structure
- **Desktop (1024px+):** Fluid 12-column layout with `2rem` outer margins and `1.25rem` column gutters. Main workspace max width caps at `1440px` centered, keeping scanning distances comfortable.
- **Tablet (768px - 1023px):** Fluid 8-column layout with `1.5rem` margins and `1rem` gutters. Sidebars fold into an off-canvas drawer or top horizontal toolbar.
- **Mobile (Below 768px):** Fluid 4-column layout with `1rem` margins and `0.75rem` gutters. Tables shift to vertical specimen cards with explicit field labels.

### Spacing Principles
- **No Box-in-Box Encapsulation:** If a page contains a task list, do not enclose the table inside a thick card nested inside a gray section card. Let the table surface sit directly on the `#F4F1EA` canvas, framed only by its single perimeter border or row rules.
- **Touch Target Integrity:** Every button, clickable chip, and menu entry must meet a strict minimum bounding box of 44x44px.

## Elevation & Depth

This system avoids layered dropshadows, synthetic 3D skeuomorphism, and blurry float effects. The interface remains flat and grounded.

### Flat Hierarchy Standard
- **Default State:** Background canvas (`#F4F1EA`) supports white surfaces (`#FFFFFF`). Separation is achieved exclusively via `1px solid #8F8678` borders.
- **No Ambient Shadows:** Cards, panels, toolbars, and input fields do not project ambient drop shadows in their default states.

### Elevated Interaction States
Shadows are permitted only for transient layers that physically hover over or occlude content:
- **Flyout Menus & Dropdowns:** `box-shadow: 0 4px 12px rgba(31, 29, 26, 0.08);` with a `1px solid #8F8678` outline.
- **Modal Dialogs:** `box-shadow: 0 12px 32px rgba(31, 29, 26, 0.14);` centered on an unblurred overlay backdrop of `rgba(31, 29, 26, 0.45)`.
- **Dragged Task / Specimen Cards:** During kanban drag operations, apply `box-shadow: 0 8px 20px rgba(31, 29, 26, 0.12);` accompanied by a 1-degree rotation to communicate manual pick-up.

## Shapes

The geometric identity is calibrated to a strict, restrained curve: softened corners that feel human and utilitarian rather than playful or bubbly.

### Uniform Geometry Rules
- **6px Strict Radius:** Applied across all cards, text inputs, selects, buttons, dropdown menus, and dialog frames.
- **Chips & Tags:** 4px to 6px radius. Full pill geometry (`rounded-full`) is prohibited for filter chips, status badges, and buttons to preserve the archival specimen card silhouette.
- **Avatars Only Exception:** User avatars representing people (`Staf A`, `Manajer A`) use circular masks (`border-radius: 9999px`) with an optional `1px solid #8F8678` ring to differentiate human entities from data cards.

## Components

### 1. Specimen Tag / Task ID
A technical, archival component used for all garment batches, tickets, and internal references.
- **Visual Spec:** IBM Plex Mono, 13px, weight 500. Display format: `[T-0042]` or `LOT-092`.
- **Styling:** Inset padding of `2px 6px`, background `#E6E1D7`, text `#1F1D1A`, border radius `4px`. Zero drop shadow.

### 2. Buttons
Action triggers with clear Indonesian operational verbs.
- **Primary Action (`"Buat task"`, `"Simpan perubahan"`, `"Masuk ke sistem"`):** Background `#3F5A3A`, text `#FFFFFF`, 1px border `#3F5A3A`, border-radius 6px, padding `10px 16px`. Hover state shifts background to `#334A2E`.
- **Secondary / Terbuka (`"Batal"`, `"Tutup"`):** Background `#FFFFFF`, text `#1F1D1A`, 1px border `#8F8678`, border-radius 6px, padding `10px 16px`. Hover state changes background to `#F4F1EA`.
- **Destructive Action (`"Hapus batch"`):** Background `#FFFFFF`, text `#9C3A22`, 1px border `#9C3A22`. Hover background tint `#FDF2F0`.
- **Focus State:** `2px solid #3F5A3A` with `2px` offset.

### 3. Status & Urgency Chips
- **Status Chips:**
  - Font: IBM Plex Sans, 13px, font-medium, border-radius 4px, padding `4px 8px`.
  - Values: `Requested` (`bg #E6E1D7, text #4A443B`), `On Progress` (`bg #E3E8DA, text #3D4F2C`), `Ready` (`bg #F1E3C8, text #6B4A12`), `Completed` (`bg #DDE5E3, text #2F4A45`).
- **Urgency Chips:**
  - Explicit word format: `Mendesak` (Urgent), `Tinggi` (High), `Sedang` (Medium), `Rendah` (Low).
  - High urgency applies text `#9C3A22` with a subtle `#9C3A22` bullet prefix. Never rely on the bullet or color alone without the accompanying text.

### 4. Form Inputs & Controls
- **Text Field / Select:** Background `#FFFFFF`, 1px border `#8F8678`, border-radius 6px, height 44px, padding `0 12px`, text `#1F1D1A` (15px).
- **Placeholder:** `#635C52` italicized or standard weight.
- **Focus:** Border color transitions to `#3F5A3A`, outline `2px solid #3F5A3A` with `2px` offset.
- **Checkboxes & Radios:** 18x18px box, border `1.5px solid #8F8678`, radius 4px (checkbox) or circular (radio). Checked fill is `#3F5A3A` with a white checkmark or center pip.

### 5. Cards & Specimen Boards
- **Kanban Card / Work Card:** Background `#FFFFFF`, 1px solid border `#8F8678`, border-radius 6px, padding `16px`. Contains Specimen Tag at top-left, Urgency label top-right, title `15px font-medium` (`Contoh: Pengecekan sampel kain batch 04`), assigned personnel chip (`Staf B`), and status chip bottom-right.
- **List / Table Rows:** Alternating rows avoid heavy stripes. Clean 1px bottom border `#8F8678`, height 52px, cell padding `8px 16px`, responsive overflow with sticky first column for task ID.

### 6. Modal Dialogs
- **Structure:** Surface `#FFFFFF`, border-radius 6px, border `1px solid #8F8678`, shadow `0 12px 32px rgba(31, 29, 26, 0.14)`.
- **Header:** Clean 18px `headline-md` title, followed by 1px horizontal rule `#8F8678`.
- **Actions Footer:** Right-aligned buttons with primary action on the right (`"Simpan perubahan"`) and secondary on the left (`"Batal"`).