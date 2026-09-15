### Fixed

- The work-browser tab took the whole window down: `WebTabSurface` named the "Show full URL" preference in an effect's dependency array above the `useState` that declares it, so rendering any browser tab threw `ReferenceError: Cannot access … before initialization` and React unmounted the app (blank page, every tab gone). A deploy from main would have carried it; a plain-browser session on main reproduces it in one click on **New browser tab**.
