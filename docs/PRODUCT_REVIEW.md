# Product review and implementation brief

Reviewed October 8, 2026 from the repository, running seeded application, and
member, advertiser, administrator, and public acquisition journeys. This is a
heuristic product review; it does not replace interviews or usability research.

## Findings and priorities

| Priority | Finding and user impact | Business value | Effort | Dependencies | Decision |
| --- | --- | --- | --- | --- | --- |
| P0 | Admin and advertiser pages inherit the 32rem feed column. Public marketing overlays the app, leaving duplicate navigation in the accessibility tree. Desktop admin access is missing. | Make business operations usable and discoverable | M | Existing shell and role data | Implement |
| P0 | Advertising registration, login, and plan selection lose the user's destination and selected package. | Reduce acquisition and onboarding drop-off | M | Auth actions, onboarding, packages | Implement |
| P0 | A unique ledger reference permits only one manual adjustment per administrator/company. Credit checkout has no clear availability state. | Restore staff credit operations and billing trust | S | Ledger and checkout configuration | Implement |
| P1 | Reporting has no date controls; one chart combines incomparable user and impression magnitudes. No keyboard-readable series or revenue chart. | Help advertisers and staff make budget and growth decisions | M | Existing range-aware analytics APIs | Implement |
| P1 | Social publishing requires an externally hosted image URL; ad service failure also hides otherwise available social posts. | Improve content creation and core platform resilience | M | Existing media storage, media ownership, additive post reference | Implement |
| P1 | Inputs frequently have placeholders without labels, suppressed focus outlines, and unannounced submission states. | Make common tasks usable with keyboard and assistive technology | S/M | Shared fields and affected forms | Implement within selected journeys |
| P1 | Landing preview contains fictional balances, results, and bars with no sample label. Delivery ranking is described as a monetary CPM bid although credits are committed upfront. | Set accurate customer expectations | S | Public aggregate metrics, current funding model | Implement truthful copy and preview |
| P1 | Advertisers cannot edit campaign creative or pause/resume/cancel their campaigns; staff can alter state but without a full audit trail. | Campaign management and operational accountability | M/L | Ownership checks, lifecycle/refund policy | Backlog |
| P1 | Card checkout, verified webhook settlement, refunds, and metered delivery quotas are absent. | Production monetization | L | Payment-provider choice, durable settlement and quota design | Backlog; retain explicit checkout availability |
| P1 | Public posting lacks reporting, moderation, blocking, and accessible content governance. | Trust and retention | L | Moderation policy, staff queues, content lifecycle | Backlog |
| P2 | Social experience lacks likes, replies, following, notifications, search, and pagination; user interaction analytics are not yet defined. | Repeat engagement and social discovery | L | Interaction data model, activity instrumentation | Backlog; do not invent metrics |
| P2 | Expo application is an unchanged starter screen. It cannot perform any platform journey. | Native distribution | L | Mobile authentication, navigation, device testing | Backlog; responsive web first |
| P2 | No research-backed pricing, targeting consent flow, account recovery, or multi-user company access. | Conversion, confidence, and larger business accounts | M/L | Policy and research, email provider, membership model | Backlog |

P0 restores broken or misleading paths. P1 completes common tasks using existing
infrastructure. Larger features remain explicit follow-on work because they
require new product policies or provider decisions. Selected work does not
change the existing prepaid campaign commitment model.

## Engineering assignments

Owner for the following implementation tasks: Staff Software Engineer (this
work session). Complete the tasks sequentially with individual commits.

### UX-01 — Responsive navigation and accessible route shells

Requirements: give social routes their focused reading layout, business routes
the available workspace width, and the advertising landing its own normal-flow
layout. Provide labeled role-aware navigation at mobile and desktop sizes, a
skip link, visible focus, and reliable profile/theme controls.

Acceptance criteria:

- At 375px and 1440px, admin/advertiser content fits the page; tables may scroll
  within their container without creating horizontal document overflow.
- Signed-in admins can reach Administration from both navigation sizes.
- Marketing renders one navigation and one main landmark; the hidden social
  shell is absent from its keyboard and accessibility tree.
- Keyboard focus is visible; skip navigation reaches main content.

### UX-02 — Preserve advertising intent through authentication

Requirements: carry an allow-listed return destination through registration,
login, switching forms, and advertiser activation. Carry an available package
selection into the credit panel. Explain the company/account setup step.

Acceptance criteria:

- A public package CTA reaches registration, then advertiser onboarding, then
  the selected package rather than Home.
- Switching sign-in/sign-up preserves the same destination.
- Arbitrary external or protocol-relative destinations cannot be redirected to.
- Members still sign in to Home by default and retain their social capabilities.

### UX-03 — Trustworthy reporting and billing operations

Requirements: add URL-based date filters to admin/advertiser analytics, reuse
backend validation, and display separate, accessible daily activity, delivery,
and revenue series. Remove fictional live metrics from public previews. Show
checkout availability before submission. Correct repeated admin adjustments.

Acceptance criteria:

- Valid 1–90-day ranges are sent to the real analytics APIs and reflected in
  charts; malformed/reversed/oversized ranges show a recoverable message.
- All-zero data renders an empty state; charts offer labeled data tables and
  do not depict fake nonzero bars.
- Revenue is formatted using cents; credits remain integer units.
- Two distinct adjustments by one admin to one company both succeed, retain
  audit actor/reason, and reconcile to the company balance on both DB providers.
- Disabled checkout is disclosed and cannot be submitted through its UI.
- Public values are live aggregates or clearly labeled illustrations; campaign
  ranking is not presented as an implemented CPM settlement price.

### UX-04 — Publish images and keep the social feed available

Requirements: add file upload/preview to publishing using existing local/S3/R2
media infrastructure; resolve media URLs from the authenticated owner in the
backend; reference images durably from posts; preserve text and show actionable
errors. Handle social and ad reads independently.

Acceptance criteria:

- A member can upload a supported image, publish it, and see it on the feed and
  profile after reload on PostgreSQL or SQLite.
- Another user cannot publish a media reference they do not own.
- Media referenced by a post is protected from deletion; unused media can be
  deleted using the existing API.
- Oversized/unsupported files are rejected before upload; failed submission
  keeps the user's text and exposes a readable status.
- Ads failing to load does not hide fetched social posts.

## Validation and rollout

Run backend/provider contract tests and race checks, web lint/type checks and
production build, plus browser checks for the public acquisition journey,
member publishing, advertiser reporting, and staff operations. Use isolated
development data. Record completion evidence and remaining limitations here.
