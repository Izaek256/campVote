# Admin User Management

**Issue:** #20 (Epic: Complete SRS Implementation — Priority 1.2)  
**Branch:** `feature/20-registration-and-user-management`  
**SRS References:** FR-AUTH-06, FR-AUTH-07  
**Status:** Implemented

---

## Summary

The admin dashboard now has a fully functional User Management page at `/dashboard/resources/users`.

Administrators can:
- View all user accounts with name, email, role, status, and email-verified state
- Search by name or email
- Filter by role and account status (active / suspended)
- Create new user accounts (with role assignment)
- Edit existing accounts (name, email, role, password, active flag)
- Delete accounts (soft-delete; recoverable from Deleted Accounts)
- Bulk-activate or bulk-suspend multiple users

All operations call the existing backend endpoints:
- `GET /api/v1/users` — list
- `POST /api/v1/users` — create
- `PUT /api/v1/users/:id` — update
- `DELETE /api/v1/users/:id` — soft-delete

Role changes made here are reflected immediately on the user's next request; no token refresh is required because roles are re-checked server-side on every request.

---

## Files Changed / Created

### New page
| File | Description |
|---|---|
| `apps/admin/app/(dashboard)/resources/users/page.tsx` | User management page |

### Admin infrastructure (all new — these fix the missing design-system layer)
| File | Description |
|---|---|
| `apps/admin/lib/api-client.ts` | Axios-based API client with CSRF and cookie support |
| `apps/admin/lib/api-core.ts` | `getApiErrorMessage` helper |
| `apps/admin/lib/icons.ts` | Lucide-react icon re-exports |
| `apps/admin/components/chrome/PageHeader.tsx` | Page title + action slot |
| `apps/admin/components/ui/button.tsx` | Button + `buttonClasses` (object and positional call forms) |
| `apps/admin/components/ui/confirm-modal.tsx` | Accessible `<dialog>`-based confirmation modal |
| `apps/admin/components/ui/input.tsx` | Input + `inputClasses` |
| `apps/admin/components/providers.tsx` | React Query provider for root layout |
| `apps/admin/components/profile/delete-account-dialog.tsx` | Self-delete confirmation |
| `apps/admin/components/profile/active-sessions.tsx` | Session list + revocation |
| `apps/admin/components/profile/two-factor-card.tsx` | TOTP status card |
| `apps/admin/hooks/use-auth.ts` | `useMe` hook |
| `apps/admin/hooks/use-profile.ts` | `useUpdateProfile`, `useChangePassword` |

### Package changes
| File | Description |
|---|---|
| `apps/admin/package.json` | Added missing dependencies: `@tanstack/react-query`, `axios`, `lucide-react`, `sonner`, `zod`, `react-hook-form`, `@hookform/resolvers`, `@repo/shared` |
| `apps/admin/tsconfig.json` | Added `@repo/shared` path alias |
| `apps/admin/app/layout.tsx` | Wrapped body in `<Providers>` for React Query |
| `packages/shared/package.json` | Created (was missing) |
| `packages/shared/schemas/user.ts` | Added `PersonalInfoSchema`, `ProfessionalInfoSchema` |
| `packages/shared/schemas/index.ts` | Created barrel export |
| `packages/shared/types/index.ts` | Created barrel export |
| `packages/shared/index.ts` | Created root barrel export |

---

## Acceptance Criteria Status

| Criterion | Status |
|---|---|
| Admins can create student/staff accounts | ✅ |
| Admins can assign roles and permissions | ✅ |
| User list is searchable and filterable | ✅ |
| Deletion has confirmation dialog | ✅ |
| Role changes reflected immediately | ✅ |
| Bulk suspend/unsuspend multiple users | ✅ |

---

## Authorization Note

The User Management page is only accessible to users with the `ADMIN` role. This is enforced server-side on all `/api/v1/users` endpoints. The admin dashboard itself does not currently enforce an auth guard at the layout level — that is deferred work documented in the epic.
