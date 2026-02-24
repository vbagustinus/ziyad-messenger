# Monorepo Platform Index

Root repo disederhanakan. Kepemilikan dokumen dan tooling sekarang ada di masing-masing platform.

- [Backend docs](backend/docs/README.md)
- [Frontend docs](frontend/docs/README.md)
- [Client docs](clients/docs/README.md)
- [Backend operations docs](backend/docs/operations/README.md)
- [Backend role charter](backend/docs/project/platform-role.md)
- [Frontend role charter](frontend/docs/project/platform-role.md)
- [Client role charter](clients/docs/project/platform-role.md)

## Working Entry Points

- Backend deploy commands: `make -f backend/deploy/Makefile <target>` atau `cd backend/deploy && make <target>`
- Frontend deploy commands: `make -f frontend/admin-dashboard/deploy/Makefile <target>`
- Go workspace backend: `cd backend` lalu jalankan command Go dari sana
