import { cpSync, existsSync, mkdirSync } from "node:fs";
import { spawn } from "node:child_process";
import { join } from "node:path";

const appRoot = process.cwd();
const standaloneRoot = join(appRoot, ".next", "standalone", "apps", "web");
const server = join(standaloneRoot, "server.js");

if (!existsSync(server)) {
  throw new Error("Standalone build not found. Run `bun run build` before starting the web app.");
}

const staticSource = join(appRoot, ".next", "static");
const staticTarget = join(standaloneRoot, ".next", "static");
mkdirSync(staticTarget, { recursive: true });
cpSync(staticSource, staticTarget, { recursive: true });

const publicSource = join(appRoot, "public");
if (existsSync(publicSource)) cpSync(publicSource, join(standaloneRoot, "public"), { recursive: true });

const child = spawn(process.execPath, [server], { env: process.env, stdio: "inherit" });
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => child.kill(signal));
child.on("exit", (code, signal) => process.exit(signal ? 1 : (code ?? 1)));
