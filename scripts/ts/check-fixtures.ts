import { readdir } from "node:fs/promises";
import { resolve } from "node:path";

async function main() {
  const fixturesRoot = resolve(process.cwd(), "data/fixtures");
  const entries = await readdir(fixturesRoot, { withFileTypes: true });
  const fixtureCount = entries.filter((entry) => entry.isDirectory()).length;
  console.log(`fixtures: ${fixtureCount}`);
}

main().catch((error: unknown) => {
  console.error(error);
  process.exit(1);
});
