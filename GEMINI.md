# FlashApp - Context & Development Rules

## Project Overview
FlashApp is a lightweight AI H5 Generation and Hosting platform prototype. 
- **Backend:** Go (Monolithic service, zero-dependency JSON store fallback, planned SQLite).
- **Frontend:** Vanilla HTML, CSS, and JS (No heavy frameworks).
- **Features:** Auth, AI stream generation of HTML5 apps, real-time iframe preview, auto-saving.

## Building and Running
- `go run ./src/cmd/flashapp`
- Or use `.\run.ps1` / `.\run.bat`
- Default port: `:18080`
- Config via env vars (e.g., `FLASHAPP_USE_MOCK=true`, `FLASHAPP_LLM_API_KEY`).

## Mandatory Development Rules

### 1. 提交与注释规范 (NEW)
- **提交信息**: 所有的 Git 提交信息必须使用 **中文** 编写，并遵循常规的 Git 提交格式（如 `feat:`, `fix:`, `refactor:` 等）。
- **代码注释**: AI 生成的代码（HTML, CSS, JS）必须包含清晰的 **中文注释**，解释关键逻辑、样式块或交互点。
- **临时文件**: 严禁提交 `temp/` 目录下的任何文件。该目录仅用于存放开发过程中的临时截图、日志或测试数据。

### 2. 文件修改安全性 (CRITICAL)
- **Single-Turn Limit:** NEVER execute multiple `replace` calls on the SAME file within a single conversational turn. Sequential modifications must span multiple turns.
- **Precision:** Perform an exact read of the target code block (including whitespace) before any `replace` to avoid offset errors and dangling tokens.
- **Verification:** Always verify syntax and structural integrity after modifying code.
- **Major Refactors:** For disjointed blocks in large files, prefer `write_file` to rewrite the document entirely.

### 2. Backend Structure
- All Go source files MUST live under `src/`. No `.go` files in the repository root.
- Executables live under `src/cmd/`. Shared logic under `src/internal/`.

### 3. Frontend Structure
- Primary entry point is `static/index.html` (Chat Interface).
- Authentication (Login/Register) is handled via a shared modal component.
- **Shared code:** `static/assets/js/shared/` and `static/assets/css/shared/`.
- **Page code:** `static/assets/js/pages/` and page-specific CSS in `static/assets/css/`.
- NEVER embed large `<style>` or `<script>` blocks inside HTML.

### 4. UI Layout Stability
- Console pages must use a viewport-bounded shell with internal scroll containers.
- History lists, chat streams, and previews MUST NOT push fixed footers or composers out of view.
- Previews must not introduce horizontal overflow.

### 5. File Size Limits
- Hard limit: Any file over 1000 lines MUST be split.
- Frontend files (HTML, CSS, JS) should generally stay below 500 lines. Split earlier if responsibilities mix.
