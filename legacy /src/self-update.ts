import { chmodSync, copyFileSync, renameSync, unlinkSync, writeFileSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';

const REPO = 'sethcarney/skills';
const RELEASES_API = `https://api.github.com/repos/${REPO}/releases/latest`;

interface GitHubAsset {
  name: string;
  browser_download_url: string;
}

interface GitHubRelease {
  tag_name: string;
  assets: GitHubAsset[];
}

function getBinaryAssetName(): string | null {
  const { platform, arch } = process;
  if (platform === 'linux' && arch === 'x64') return 'skills-linux-x64';
  if (platform === 'linux' && arch === 'arm64') return 'skills-linux-arm64';
  if (platform === 'darwin' && arch === 'x64') return 'skills-macos-x64';
  if (platform === 'darwin' && arch === 'arm64') return 'skills-macos-arm64';
  if (platform === 'win32' && arch === 'x64') return 'skills-windows-x64.exe';
  return null;
}

// When compiled with Bun, the Bun global is available; in Node/npx it is not.
function isRunningAsBinary(): boolean {
  return typeof (globalThis as Record<string, unknown>)['Bun'] !== 'undefined';
}

function isNewer(latest: string, current: string): boolean {
  const parse = (v: string) => v.split('.').map(Number);
  const [l, c] = [parse(latest), parse(current)];
  for (let i = 0; i < 3; i++) {
    const diff = (l[i] ?? 0) - (c[i] ?? 0);
    if (diff > 0) return true;
    if (diff < 0) return false;
  }
  return false;
}

export async function runSelfUpdate(currentVersion: string): Promise<void> {
  const RESET = '\x1b[0m';
  const DIM = '\x1b[38;5;102m';
  const TEXT = '\x1b[38;5;145m';

  console.log(`${DIM}Checking for updates...${RESET}`);

  let release: GitHubRelease;
  try {
    const res = await fetch(RELEASES_API, {
      headers: { 'User-Agent': `skills-cli/${currentVersion}` },
    });
    if (!res.ok) throw new Error(`GitHub API returned ${res.status}`);
    release = (await res.json()) as GitHubRelease;
  } catch (err) {
    console.error(
      `Failed to check for updates: ${err instanceof Error ? err.message : String(err)}`
    );
    process.exit(1);
  }

  const latestVersion = release.tag_name.replace(/^v/, '');

  if (!isNewer(latestVersion, currentVersion)) {
    console.log(`${TEXT}Already up to date${RESET} ${DIM}(${currentVersion})${RESET}`);
    return;
  }

  console.log(
    `${TEXT}New version available:${RESET} ${latestVersion} ${DIM}(current: ${currentVersion})${RESET}`
  );
  console.log();

  if (!isRunningAsBinary()) {
    console.log(
      `${DIM}You are running skills via npx or npm. To get self-updates, install the binary:${RESET}`
    );
    console.log();
    console.log(
      `  ${TEXT}curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sh${RESET}`
    );
    console.log();
    return;
  }

  const assetName = getBinaryAssetName();
  if (!assetName) {
    console.error(`Unsupported platform: ${process.platform}/${process.arch}`);
    console.error(`Download manually from: https://github.com/${REPO}/releases/latest`);
    process.exit(1);
  }

  const asset = release.assets.find((a) => a.name === assetName);
  if (!asset) {
    console.error(`Binary for your platform (${assetName}) not found in release ${latestVersion}.`);
    console.error(`Check: https://github.com/${REPO}/releases/tag/v${latestVersion}`);
    process.exit(1);
  }

  console.log(`${DIM}Downloading ${assetName}...${RESET}`);
  const downloadRes = await fetch(asset.browser_download_url);
  if (!downloadRes.ok) {
    console.error(`Download failed: ${downloadRes.status}`);
    process.exit(1);
  }

  const buffer = Buffer.from(await downloadRes.arrayBuffer());
  const tmpPath = join(tmpdir(), `skills-update-${Date.now()}`);
  writeFileSync(tmpPath, buffer);

  const currentBinary = process.execPath;

  if (process.platform !== 'win32') {
    chmodSync(tmpPath, 0o755);
    try {
      renameSync(tmpPath, currentBinary);
    } catch (err: unknown) {
      if ((err as NodeJS.ErrnoException).code === 'EXDEV') {
        // Unlink the running binary first so copyFileSync can write to the path
        // without hitting ETXTBSY (text file busy). The running process retains
        // its inode and continues normally.
        unlinkSync(currentBinary);
        copyFileSync(tmpPath, currentBinary);
        unlinkSync(tmpPath);
      } else {
        throw err;
      }
    }
    console.log(`${TEXT}Updated to ${latestVersion} successfully.${RESET}`);
    console.log(
      `${DIM}Restart your shell or run ${TEXT}skills --version${DIM} to confirm.${RESET}`
    );
  } else {
    // Windows cannot replace a running executable directly.
    // Write a helper batch file the user can run after exiting.
    const batchPath = join(tmpdir(), 'skills-update.bat');
    writeFileSync(
      batchPath,
      `@echo off\r\ntimeout /t 1 /nobreak > NUL\r\nmove /y "${tmpPath}" "${currentBinary}" > NUL\r\ndel "%~f0"\r\n`
    );
    console.log(`${TEXT}Downloaded ${latestVersion} to:${RESET} ${tmpPath}`);
    console.log(`${DIM}To apply the update, run after exiting this process:${RESET}`);
    console.log(`  ${TEXT}${batchPath}${RESET}`);
  }
}
