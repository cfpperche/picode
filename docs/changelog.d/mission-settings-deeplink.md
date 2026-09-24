### Changed

- A Pi Settings workspace deep link (`#/clis/pi/settings?workspaceId=…`) keeps that workspace through a reload, the layer switcher and the Settings/Keyboard tabs, and names the folder on the project layer even with no agent selected. A missing or unknown id is an error with a way back to Global settings — it does not silently open another workspace or pick an agent.
