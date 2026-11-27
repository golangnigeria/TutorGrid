import { z } from "zod";

const envVarsSchema = z.object({
  VITE_API_URL: z.string().default(""),
  VITE_ENV: z.enum(["production", "development", "local"]).default("local"),
});

const parseResult = envVarsSchema.safeParse(import.meta.env);

const envVars = parseResult.success
  ? parseResult.data
  : { VITE_API_URL: "", VITE_ENV: "local" as const };

export const ENV = envVars.VITE_ENV;
export const API_URL = envVars.VITE_API_URL;
