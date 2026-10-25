import { get } from "../../utils/request";

// Skill信息
export interface SkillInfo {
  name: string;
  description: string;
}

/** Locates a shared agent's source workspace, same parameters as a chat request. */
export interface AgentScope {
  agentId?: string;
  /** Set only for a shared agent; left empty for this workspace's own agents. */
  sourceTenantId?: string | number;
}

// Lists the skills executable on the current sandbox config. Without a
// sandboxConfigId, or when skills_available is false, the caller should hide or
// disable the skills UI.
//
// Pass the agent for a shared one: its skills are installed on a sandbox config
// in the OWNER's workspace, so looking them up in the caller's returns nothing.
// The backend then reads the config off that agent and ignores sandboxConfigId.
export function listSkills(sandboxConfigId?: string, agent?: AgentScope) {
  const params: Record<string, string> = {};
  if (sandboxConfigId) params.sandbox_config_id = sandboxConfigId;
  if (agent?.agentId && agent?.sourceTenantId) {
    params.agent_id = agent.agentId;
    params.agent_source_tenant_id = String(agent.sourceTenantId);
  }
  return get<{ data: SkillInfo[]; skills_available?: boolean }>('/api/v1/skills', { params });
}
