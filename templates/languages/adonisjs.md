## AdonisJS-Specific Rules

### UUID Models: `selfAssignPrimaryKey = true`

```typescript
import { randomUUID } from 'node:crypto'
import { BaseModel, beforeCreate, column } from '@adonisjs/lucid/orm'

export default class MyModel extends BaseModel {
  static selfAssignPrimaryKey = true // REQUIRED for UUID models

  @column({ isPrimary: true })
  declare id: string

  @beforeCreate()
  static assignUuid(model: MyModel) {
    model.id = randomUUID()
  }
}
```

### Serialize Before Inertia Render

Never pass Lucid models directly to `inertia.render()`. Always use `.serialize()` or manual mapping. DateTime fields need `.toISO()`.

```typescript
return inertia.render('page', {
  items: items.map((item) => ({
    id: item.id,
    name: item.name,
    createdAt: item.createdAt.toISO(),
  })),
})
```

### SQLite-Compatible Queries

Use `whereRaw('LOWER(col) LIKE ?', [...])` instead of `whereILike()`.

### Never Auto-Run Migrations

Never add migrations to Dockerfile CMD, startup scripts, or npm hooks.

### Code Review Quick Reference

| Area | Check |
|------|-------|
| Models | `selfAssignPrimaryKey = true` for UUIDs |
| Models | `serializeAs: null` for sensitive fields |
| Controllers | Data serialized before Inertia render |
| Controllers | DateTime converted with `.toISO()` |
| Database | Migrations work with SQLite |
