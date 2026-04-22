import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..', '..');
const sourcePath = path.join(__dirname, 'sect_map_short_session.v1.json');
const clientOutputPath = path.join(
  repoRoot,
  'client',
  'my-immortal-sect',
  'assets',
  'scripts',
  'app',
  'sect-map-shared-config.generated.ts',
);

const source = JSON.parse(fs.readFileSync(sourcePath, 'utf8'));
const output = `/* Auto-generated from shared/configs/m1/sect_map_short_session.v1.json. Do not edit by hand. */
export const SECT_MAP_SHARED_SHORT_SESSION_CONFIG = ${JSON.stringify(source, null, 4)} as const;
`;

fs.writeFileSync(clientOutputPath, output);
console.log(clientOutputPath);
