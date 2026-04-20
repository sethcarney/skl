import * as p from '@clack/prompts';
import pc from 'picocolors';
import { readLocalLock } from './local-lock.ts';
import { runAdd } from './add.ts';
import { selectAgentsInteractive } from './add.ts';
import { runSync, parseSyncOptions } from './sync.ts';
import { getUniversalAgents } from './agents.ts';
import type { AgentType } from './types.ts';

interface InstallOptions {
  agent?: string[];
  yes?: boolean;
}

function parseInstallOptions(args: string[]): InstallOptions {
  const options: InstallOptions = {};

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '-y' || arg === '--yes') {
      options.yes = true;
    } else if (arg === '-a' || arg === '--agent') {
      options.agent = options.agent ?? [];
      i++;
      let next = args[i];
      while (i < args.length && next && !next.startsWith('-')) {
        options.agent.push(next);
        i++;
        next = args[i];
      }
      i--;
    }
  }

  return options;
}

/**
 * Install all skills from the local skills-lock.json.
 * Groups skills by source and calls `runAdd` for each group.
 *
 * Without --yes, shows an interactive agent selection prompt so the
 * user can choose which agents to install to beyond the universal set.
 * Universal agents (.agents/skills/) are always included and shown as locked.
 *
 * node_modules skills are handled via experimental_sync.
 */
export async function runInstallFromLock(args: string[]): Promise<void> {
  const cwd = process.cwd();
  const options = parseInstallOptions(args);
  const lock = await readLocalLock(cwd);
  const skillEntries = Object.entries(lock.skills);

  if (skillEntries.length === 0) {
    p.log.warn('No project skills found in skills-lock.json');
    p.log.info(
      `Add project-level skills with ${pc.cyan('npx skills add <package> --project')} or ${pc.cyan('npx skills add <package>')} (without ${pc.cyan('-g')})`
    );
    return;
  }

  // Determine target agents
  let targetAgents: AgentType[];

  if (options.agent && options.agent.length > 0) {
    // Explicit --agent flag: use as-is
    targetAgents = options.agent as AgentType[];
  } else if (options.yes) {
    // Non-interactive: fall back to universal agents only (safe default)
    targetAgents = getUniversalAgents();
  } else {
    // Interactive: let the user pick agents; universal agents are locked/always included
    const selected = await selectAgentsInteractive({ global: false });

    if (p.isCancel(selected)) {
      p.cancel('Installation cancelled');
      process.exit(0);
    }

    targetAgents = selected as AgentType[];
  }

  // Separate node_modules skills from remote/local skills
  const nodeModuleSkills: string[] = [];
  const bySource = new Map<string, { sourceType: string; skills: string[] }>();

  for (const [skillName, entry] of skillEntries) {
    if (entry.sourceType === 'node_modules') {
      nodeModuleSkills.push(skillName);
      continue;
    }

    const installSource = entry.ref ? `${entry.source}#${entry.ref}` : entry.source;
    const existing = bySource.get(installSource);
    if (existing) {
      existing.skills.push(skillName);
    } else {
      bySource.set(installSource, {
        sourceType: entry.sourceType,
        skills: [skillName],
      });
    }
  }

  const remoteCount = skillEntries.length - nodeModuleSkills.length;
  if (remoteCount > 0) {
    p.log.info(
      `Restoring ${pc.cyan(String(remoteCount))} skill${remoteCount !== 1 ? 's' : ''} from skills-lock.json`
    );
  }

  // Install remote/local skills grouped by source
  for (const [source, { skills }] of bySource) {
    try {
      await runAdd([source], {
        skill: skills,
        agent: targetAgents,
        yes: true,
      });
    } catch (error) {
      p.log.error(
        `Failed to install from ${pc.cyan(source)}: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    }
  }

  // Handle node_modules skills via sync
  if (nodeModuleSkills.length > 0) {
    p.log.info(
      `${pc.cyan(String(nodeModuleSkills.length))} skill${nodeModuleSkills.length !== 1 ? 's' : ''} from node_modules`
    );
    try {
      const { options: syncOptions } = parseSyncOptions(args);
      await runSync(args, { ...syncOptions, yes: true, agent: targetAgents });
    } catch (error) {
      p.log.error(
        `Failed to sync node_modules skills: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    }
  }
}
