# 文档保留台账（Document Retention Register）

面向档案管理员的命令行工具，用 Go 标准库实现。一次读取一条操作命令，
向标准输出返回 JSON，并把台账状态保存到本地文件。

## 运行

```
go run . --state 状态路径 --input 命令.json
```

- `--state`：台账状态文件，不存在时自动创建；成功命令后原子写回（先写
  `*.tmp` 再重命名），失败命令不改动状态。
- `--input`：包含一条命令的 JSON 文件（UTF-8，允许带 BOM）。
- 结果以 JSON 打印到标准输出；失败时输出 `{"ok":false,"error":...}` 并以
  非零状态码退出。

## 命令文件格式

所有命令形如：

```json
{ "command": "命令名", "args": { ... } }
```

日期一律为 `YYYY-MM-DD`，保留年数为正整数。

### 1. 登记类别保留年数

```json
{ "command": "register_category", "args": { "id": "tax", "retention_years": 3 } }
```

### 2. 登记文档

文档必须关联已登记类别，并给出起算日期。

```json
{ "command": "register_document",
  "args": { "id": "d1", "category": "tax", "start_date": "2020-02-29" } }
```

### 3. 管理法务保留

每份文档可有多个具名保留，全部解除后才恢复销毁资格。

```json
{ "command": "add_hold",     "args": { "document_id": "d1", "hold": "court-case-7" } }
{ "command": "release_hold", "args": { "document_id": "d1", "hold": "court-case-7" } }
```

### 4. 查询资格

```json
{ "command": "query", "args": { "document_id": "d1", "date": "2023-02-28" } }
```

输出包含 `expiry_date`（到期日）与 `status`。状态优先级依次为：

- `destroyed` 已销毁（记录仍可查询，且禁止再修改）
- `held` 被保留（存在至少一个具名法务保留）
- `not_due` 未到期
- `eligible` 可销毁

### 5. 预览销毁清单

生成带唯一编号的清单（如 `L0001`），列出该日期全部可销毁文档。清单持久
化到状态中，可在进程重启后继续确认；没有可销毁文档时拒绝生成空清单。

```json
{ "command": "preview_destroy", "args": { "date": "2023-02-28" } }
```

### 6. 确认销毁

```json
{ "command": "confirm_destroy",
  "args": { "list_id": "L0001", "confirm_date": "2023-03-01" } }
```

规则：

- 确认日期不得早于预览日期。
- 只能引用清单编号，不能加入清单外的文档。
- 确认时对清单逐项重新检查资格：任一项已销毁、被保留或未到期，整批拒绝
  且状态不变。
- 空清单、已确认清单、未知清单均拒绝。
- 成功后记录确认日期并把整批标记为已销毁；不会删除任何真实文件。

## 到期日规则

到期日 = 起算日期加对应整年数。若起算日为 2 月 29 日且目标年不是闰年，
到期日取 2 月 28 日（如 2020-02-29 + 1 年 = 2021-02-28；+ 4 年 =
2024-02-29）。到期日当天即视为可销毁。

## 拒绝条件（失败不改状态）

- 重复的类别或文档标识、重复/不存在的保留名。
- 未知类别、未知文档、未知清单、未知命令。
- 非法日期、非正整数保留年数、缺失或类型错误的字段。
- 对已销毁文档做修改。

## 重启后继续确认清单的示例

第一次运行生成预览（进程随后可结束）：

```
go run . --state ./register.json --input preview.json
```

`preview.json`：

```json
{ "command": "preview_destroy", "args": { "date": "2023-02-28" } }
```

输出：

```json
{ "ok": true, "list_id": "L0001", "preview_date": "2023-02-28",
  "document_ids": ["d1"], "count": 1 }
```

之后即使重启程序，清单仍保存在 `register.json` 中，直接凭编号确认：

```
go run . --state ./register.json --input confirm.json
```

`confirm.json`：

```json
{ "command": "confirm_destroy",
  "args": { "list_id": "L0001", "confirm_date": "2023-03-01" } }
```

## 自测

```
go test ./...
```

测试覆盖闰日到期、状态优先级、多保留、预览/确认整批拒绝、日期顺序、重复
标识、未知对象与已销毁文档不可修改等场景。
