# RBAC Matrix

## Roles

| Role | Description | Typical use |
|------|-------------|-------------|
| **super_admin** | Full system control; bypass department restrictions | Platform operators |
| **admin** | Manage users, roles, channels, and org settings within scope | Department or org admins |
| **moderator** | Manage channel membership and content within assigned channels | Channel moderators |
| **member** | Use channels and send messages within allowed channels | Standard users |
| **auditor** | Read-only access to audit logs and compliance data | Compliance, security |
| **guest** | Limited read/write in explicitly allowed channels | Contractors, temporary |
| **service** | Machine identity for backend services (e.g. Messaging Router) | Service accounts |

## Permissions (Resource:Action)

| Permission | Description |
|------------|-------------|
| **user:create** | Create user accounts |
| **user:read** | Read user profile and list |
| **user:update** | Update user profile and role |
| **user:delete** | Delete or disable user |
| **role:assign** | Assign/change roles for users |
| **channel:create** | Create channels |
| **channel:read** | Read channel metadata and membership |
| **channel:update** | Update channel settings |
| **channel:delete** | Delete or archive channel |
| **channel:join** | Join channel |
| **channel:leave** | Leave channel |
| **channel:manage_members** | Add/remove members, set moderator |
| **message:send** | Send message to channel |
| **message:read** | Read messages in channel |
| **message:delete** | Delete message (own or as moderator) |
| **message:pin** | Pin/unpin messages |
| **file:upload** | Upload file to channel or DM |
| **file:download** | Download file |
| **file:delete** | Delete file |
| **audit:read** | Read audit log |
| **audit:export** | Export audit log (compliance) |
| **system:config** | Read/update system configuration |
| **system:admin** | Admin console access |
| **broadcast:send** | Send system-wide or emergency broadcast |
| **device:register** | Register device for account |
| **device:revoke** | Revoke device binding |

## Role → Permission Matrix

| Permission | super_admin | admin | moderator | member | auditor | guest | service |
|------------|:-----------:|:-----:|:---------:|:------:|:-------:|:-----:|:-------:|
| user:create | ✓ | ✓* | — | — | — | — | — |
| user:read | ✓ | ✓ | ✓* | ✓* | ✓ | ✓* | — |
| user:update | ✓ | ✓* | — | self | — | — | — |
| user:delete | ✓ | ✓* | — | — | — | — | — |
| role:assign | ✓ | ✓* | — | — | — | — | — |
| channel:create | ✓ | ✓ | — | ✓* | — | — | — |
| channel:read | ✓ | ✓ | ✓ | ✓ | ✓* | ✓* | ✓* |
| channel:update | ✓ | ✓ | ✓* | — | — | — | — |
| channel:delete | ✓ | ✓ | ✓* | — | — | — | — |
| channel:join | ✓ | ✓ | ✓ | ✓ | — | ✓* | — |
| channel:leave | ✓ | ✓ | ✓ | ✓ | — | ✓ | — |
| channel:manage_members | ✓ | ✓ | ✓ | — | — | — | — |
| message:send | ✓ | ✓ | ✓ | ✓ | — | ✓* | ✓* |
| message:read | ✓ | ✓ | ✓ | ✓ | — | ✓* | ✓* |
| message:delete | ✓ | ✓ | ✓ | ✓* | — | — | — |
| message:pin | ✓ | ✓ | ✓ | — | — | — | — |
| file:upload | ✓ | ✓ | ✓ | ✓ | — | ✓* | ✓* |
| file:download | ✓ | ✓ | ✓ | ✓ | — | ✓* | ✓* |
| file:delete | ✓ | ✓ | ✓ | ✓* | — | — | — |
| audit:read | ✓ | ✓* | — | — | ✓ | — | ✓* |
| audit:export | ✓ | ✓* | — | — | ✓ | — | — |
| system:config | ✓ | ✓* | — | — | — | — | — |
| system:admin | ✓ | ✓ | — | — | — | — | — |
| broadcast:send | ✓ | ✓* | — | — | — | — | — |
| device:register | ✓ | ✓ | — | self | — | — | — |
| device:revoke | ✓ | ✓ | — | self | — | — | — |

- **✓** = full grant  
- **✓*** = scoped (e.g. within department, or own resource only)  
- **self** = only own profile/device  
- **—** = no permission  

Scoping rules (e.g. admin only in same department) are enforced in the Auth and Channel services according to **department** and **resource ownership**.

## Resource Hierarchy (for scoping)

- **System** → **Organization** → **Department** → **Channel** → **Message/File**
- **admin** scope: organization or department; **moderator** scope: per channel.

## Emergency and Broadcast

- **broadcast:send** is restricted to **super_admin** and optionally **admin** (e.g. for emergency alerts).
- Emergency messages can be delivered with higher QoS and optional override of “busy”/DND.
