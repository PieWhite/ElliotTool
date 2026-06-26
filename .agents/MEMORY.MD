# Description
Manages the long-term memory, technical decisions, and learned lessons for this specific project. Triggers when a complex bug is resolved, a new architectural decision is made, or when the user asks to save a lesson.

# Instructions
- **The Memory Bank:** The file `.agents/MEMORY.md` is your long-term memory storage. 
- **Read Before Acting:** When solving complex bugs related to Go or Vue in this project, briefly scan `.agents/MEMORY.md` first to ensure you are not repeating past mistakes or violating project-specific workarounds.
- **Proactive Logging:** If we spend significant time debugging a specific issue (e.g., a quirk in Polygon.io's API, a specific reactivity issue in Vue, or a Go concurrency gotcha), you MUST proactively suggest adding a brief technical summary to `.agents/MEMORY.md`.
- **Formatting:** When writing to `.agents/MEMORY.md`, use clear headings (e.g., `### Vue Chart Reactivity (Date)`), explain the *Problem*, and document the *Solution/Decision*.