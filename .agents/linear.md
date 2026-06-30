# Linear Project Reference

## Project

- **Name:** Discogs Collection Manager
- **URL:** https://linear.app/automatta-tech/project/discogs-collection-manager-1c0eb5085335
- **Team:** Automatta (`ATMT`)

## Key Documents

- **PRD:** https://linear.app/automatta-tech/document/prd-discogs-collection-manager-4d2e202ef767

## Issues

| ID | Title | Status |
|----|-------|--------|
| [ATMT-70](https://linear.app/automatta-tech/issue/ATMT-70) | Project scaffold & dev tooling | Backlog |
| [ATMT-71](https://linear.app/automatta-tech/issue/ATMT-71) | User registration & login with sessions | Backlog |
| [ATMT-72](https://linear.app/automatta-tech/issue/ATMT-72) | Discogs OAuth 1.0a account connection | Backlog |
| [ATMT-73](https://linear.app/automatta-tech/issue/ATMT-73) | Barcode scanner UI | Backlog |
| [ATMT-74](https://linear.app/automatta-tech/issue/ATMT-74) | Email 2FA on login & password reset | Backlog |
| [ATMT-75](https://linear.app/automatta-tech/issue/ATMT-75) | Single-match scan confirmation flow | Backlog |
| [ATMT-76](https://linear.app/automatta-tech/issue/ATMT-76) | Conflict resolution flow | Backlog |

## Dependency Graph

```
ATMT-70  Project scaffold & dev tooling
└── ATMT-71  User registration & login with sessions
    ├── ATMT-74  Email 2FA on login & password reset
    ├── ATMT-72  Discogs OAuth 1.0a account connection ─┐
    └── ATMT-73  Barcode scanner UI ────────────────────┤
                                                        └── ATMT-75  Single-match scan confirmation flow
                                                                └── ATMT-76  Conflict resolution flow
```
