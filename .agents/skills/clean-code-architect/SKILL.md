# Description
Enforces strict SOLID principles, DRY, KISS, and general clean code standards for all architectural decisions. Triggers an ACTIVE FLAG protocol if violations are detected.

# Instructions
- **Strict SOLID Adherence:** Every piece of code must adhere to SOLID principles. Use interfaces, keep structs focused (Single Responsibility), and ensure components are open for extension but closed for modification.
- **KISS Principle (Keep It Simple, Stupid):** Prioritize the absolute simplest solution that meets the requirements. Do not over-engineer. Avoid premature optimization, unnecessary abstractions, and complex layers.
- **DRY & Code Smells:** Enforce DRY (Don't Repeat Yourself) ruthlessly. 
- **ACTIVE FLAG PROTOCOL:** If you detect a violation of SOLID, a breach of DRY, or a general Clean Code smell during generation or code scanning, you MUST immediately halt, flag the issue to the user, and propose a refactored solution.
- **Problem Solving Strategy:** For complex mathematical or architectural tasks, outline the logical steps silently or via brief bullet points before writing the code.
- **Architectural Shifts:** If a request requires breaking SOLID principles or a massive structural rewrite, outline the consequences and ask the user for confirmation before proceeding.
- **Zero Fluff & Exact Snippets:** Be concise and direct. When modifying files, output precise code blocks with surrounding lines for context. Do not output the entire file unless explicitly requested.
- **No Placeholders:** Write complete, production-ready logic. Never leave `// TODO` or `// rest of code` blocks.