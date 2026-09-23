import { useCallback, useEffect, useRef, useState } from "react";
import { createUseMissions } from "@picode/shared/client/useMissions.js";
export const useMissions = createUseMissions({ useCallback, useEffect, useRef, useState });
