#!/usr/bin/env node
/**
 * pake --iterative-build only skips DMG on macOS. On Windows, WinBuilder still
 * runs WiX MSI (light.exe), which often fails for non-ASCII product names.
 * Force tauri --no-bundle + keepBinary so cloud builds produce a raw .exe.
 */
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const cliPath = path.join(
  execSync("npm root -g", { encoding: "utf8" }).trim(),
  "pake-cli",
  "dist",
  "cli.js",
);

let source = fs.readFileSync(cliPath, "utf8");
const marker = "class WinBuilder extends BaseBuilder";
const idx = source.indexOf(marker);
if (idx < 0) {
  throw new Error("WinBuilder not found in " + cliPath);
}

const head = source.slice(0, idx);
let body = source.slice(idx);

if (!body.includes("/* pake-gui-exe-only */")) {
  const ctorNeedle = "this.options.targets = this.buildFormat;";
  const ctorIdx = body.indexOf(ctorNeedle);
  if (ctorIdx < 0) {
    throw new Error("WinBuilder constructor targets assignment not found");
  }
  body =
    body.slice(0, ctorIdx + ctorNeedle.length) +
    [
      "",
      "        /* pake-gui-exe-only */",
      "        this.options.bundle = false;",
      "        this.options.keepBinary = true;",
    ].join("\n") +
    body.slice(ctorIdx + ctorNeedle.length);

  const cmdNeedle =
    "return this.buildBaseCommand(packageManager, configPath, buildTarget);";
  const cmdIdx = body.indexOf(cmdNeedle);
  if (cmdIdx < 0) {
    throw new Error("WinBuilder getBuildCommand return not found");
  }
  body =
    body.slice(0, cmdIdx) +
    'return this.buildBaseCommand(packageManager, configPath, buildTarget) + " --no-bundle"; /* pake-gui-exe-only */' +
    body.slice(cmdIdx + cmdNeedle.length);
}

fs.writeFileSync(cliPath, head + body);
console.log("Patched WinBuilder for exe-only (--no-bundle) at", cliPath);
