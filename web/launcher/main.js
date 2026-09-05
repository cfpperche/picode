import { pickShell, shellURL } from "../shared/client/shell.js";

location.replace(shellURL(location.href, pickShell()));
