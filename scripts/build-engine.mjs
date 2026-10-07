import { execSync } from "node:child_process";

const isWindows = process.platform === "win32";

const output = isWindows
  ? "meetrec.exe"
  : "meetrec";

console.log(
  `Building recording engine for ${process.platform}...`
);

execSync(
  `go -C engine build -o ${output} .`,
  {
    stdio: "inherit",
    shell: true
  }
);

console.log(
  `Recording engine built successfully: engine/${output}`
);