import sharp from 'sharp';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __dirname = dirname(fileURLToPath(import.meta.url));
const input = join(__dirname, '..', 'shop-background.png');
const output = join(__dirname, '..', 'public', 'shop-bg.webp');

const info = await sharp(input)
  .resize({ width: 1920, withoutEnlargement: true })
  .webp({ quality: 85 })
  .toFile(output);

console.log(`Done: ${output}`);
console.log(`  ${info.width}x${info.height}, ${(info.size / 1024).toFixed(0)} KB`);
