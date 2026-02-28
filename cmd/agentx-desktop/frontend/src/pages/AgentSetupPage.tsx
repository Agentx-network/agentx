import { useState, useEffect } from "react";
import NeonButton from "../components/ui/NeonButton";
import NeonCard from "../components/ui/NeonCard";
import NeonInput from "../components/ui/NeonInput";

interface BootstrapFile {
  name: string;
  path: string;
  content: string;
  exists: boolean;
}

interface SkillEntry {
  name: string;
  source: string;
  description: string;
  path: string;
}

interface Props {
  showToast: (msg: string, type: "success" | "error") => void;
  onComplete?: () => void;
}

const fileDescriptions: Record<string, string> = {
  "SOUL.md": "Personality traits, communication style, tone, and values",
  "IDENTITY.md": "Name, tagline, purpose, capabilities, and limitations",
  "AGENTS.md": "Workflow rules, task patterns, and multi-channel behavior",
  "USER.md": "Your profile, preferences, interests, and project context",
};

export default function AgentSetupPage({ showToast, onComplete }: Props) {
  const [tab, setTab] = useState<"files" | "skills">("files");
  const [files, setFiles] = useState<BootstrapFile[]>([]);
  const [editingFile, setEditingFile] = useState<string | null>(null);
  const [editContent, setEditContent] = useState("");
  const [saving, setSaving] = useState(false);

  const [skills, setSkills] = useState<SkillEntry[]>([]);
  const [installRepo, setInstallRepo] = useState("");
  const [installing, setInstalling] = useState(false);

  const loadFiles = async () => {
    try {
      const f = await window.go.main.AgentSetupService.GetBootstrapFiles();
      setFiles(f);
    } catch { /* noop */ }
  };

  const loadSkills = async () => {
    try {
      const s = await window.go.main.AgentSetupService.ListSkills();
      setSkills(s || []);
    } catch { /* noop */ }
  };

  useEffect(() => {
    loadFiles();
    loadSkills();
  }, []);

  const handleCreateDefaults = async () => {
    try {
      await window.go.main.AgentSetupService.CreateDefaultBootstrapFiles();
      showToast("Default files created!", "success");
      loadFiles();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const handleEdit = (file: BootstrapFile) => {
    setEditingFile(file.name);
    setEditContent(file.content);
  };

  const handleSave = async () => {
    if (!editingFile) return;
    setSaving(true);
    try {
      await window.go.main.AgentSetupService.SaveBootstrapFile(editingFile, editContent);
      showToast(`${editingFile} saved!`, "success");
      setEditingFile(null);
      loadFiles();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setSaving(false);
    }
  };

  const handleInstallSkill = async () => {
    if (!installRepo.trim()) return;
    setInstalling(true);
    try {
      await window.go.main.AgentSetupService.InstallSkill(installRepo.trim());
      showToast("Skill installed!", "success");
      setInstallRepo("");
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setInstalling(false);
    }
  };

  const handleInstallBuiltins = async () => {
    setInstalling(true);
    try {
      await window.go.main.AgentSetupService.InstallBuiltinSkills();
      showToast("Builtin skills installed!", "success");
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    } finally {
      setInstalling(false);
    }
  };

  const handleRemoveSkill = async (name: string) => {
    try {
      await window.go.main.AgentSetupService.RemoveSkill(name);
      showToast(`${name} removed`, "success");
      loadSkills();
    } catch (e: any) {
      showToast(`Failed: ${e}`, "error");
    }
  };

  const hasAnyFiles = files.some((f) => f.exists);

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="text-center space-y-2">
        <h2 className="text-3xl font-bold uppercase tracking-[0.2em] text-glow-pink">
          Configure Agent
        </h2>
        <p className="text-white/35 text-sm">
          Set up your agent's identity, personality, and skills.
        </p>
      </div>

      {/* Tab switcher */}
      <div className="flex gap-2 justify-center">
        <button
          onClick={() => setTab("files")}
          className={`px-4 py-2 text-xs font-bold uppercase tracking-widest rounded-lg border-2 transition-all ${
            tab === "files"
              ? "border-neon-pink/50 bg-neon-pink/15 text-neon-pink shadow-glow-pink-sm"
              : "border-white/10 text-white/40 hover:border-white/20"
          }`}
        >
          Identity & Personality
        </button>
        <button
          onClick={() => setTab("skills")}
          className={`px-4 py-2 text-xs font-bold uppercase tracking-widest rounded-lg border-2 transition-all ${
            tab === "skills"
              ? "border-neon-cyan/50 bg-neon-cyan/15 text-neon-cyan shadow-[0_0_10px_rgba(0,255,255,0.2)]"
              : "border-white/10 text-white/40 hover:border-white/20"
          }`}
        >
          Skills
        </button>
      </div>

      {/* ── Bootstrap Files Tab ── */}
      {tab === "files" && (
        <>
          {!hasAnyFiles && (
            <NeonCard variant="purple" glow>
              <div className="text-center space-y-3">
                <p className="text-sm text-white/50">
                  No agent identity files found. Create defaults to get started.
                </p>
                <NeonButton onClick={handleCreateDefaults} size="lg" className="w-full">
                  Create Default Agent Files
                </NeonButton>
              </div>
            </NeonCard>
          )}

          {/* Editing a file */}
          {editingFile && (
            <NeonCard variant="pink" glow>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-bold uppercase tracking-widest text-neon-pink">
                    {editingFile}
                  </h3>
                  <button
                    onClick={() => setEditingFile(null)}
                    className="text-xs text-white/30 hover:text-white/60"
                  >
                    Cancel
                  </button>
                </div>
                <textarea
                  value={editContent}
                  onChange={(e) => setEditContent(e.target.value)}
                  rows={16}
                  className="w-full bg-black/40 border-2 border-neon-purple/20 rounded-lg px-4 py-3 text-sm text-white/80 font-mono leading-relaxed focus:outline-none focus:border-neon-pink/40 focus:shadow-neon-pink transition-all resize-y"
                />
                <NeonButton onClick={handleSave} disabled={saving} className="w-full">
                  {saving ? "Saving..." : "Save"}
                </NeonButton>
              </div>
            </NeonCard>
          )}

          {/* File list */}
          {!editingFile &&
            files.map((file) => (
              <NeonCard key={file.name} variant="purple">
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <h3 className="text-sm font-bold uppercase tracking-widest text-white/80">
                      {file.name}
                    </h3>
                    <p className="text-xs text-white/30 mt-0.5">
                      {fileDescriptions[file.name] || "Agent configuration file"}
                    </p>
                    {file.exists && (
                      <p className="text-[10px] text-neon-green/40 mt-1 font-mono">
                        {file.content.split("\n").length} lines
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    {file.exists ? (
                      <NeonButton
                        onClick={() => handleEdit(file)}
                        variant="ghost"
                        size="sm"
                      >
                        Edit
                      </NeonButton>
                    ) : (
                      <NeonButton
                        onClick={() => {
                          setEditingFile(file.name);
                          setEditContent(`# ${file.name.replace(".md", "")}\n\n`);
                        }}
                        size="sm"
                      >
                        Create
                      </NeonButton>
                    )}
                  </div>
                </div>
              </NeonCard>
            ))}

          {hasAnyFiles && !editingFile && (
            <NeonButton
              onClick={handleCreateDefaults}
              variant="ghost"
              size="sm"
              className="w-full"
            >
              Reset to Defaults
            </NeonButton>
          )}
        </>
      )}

      {/* ── Skills Tab ── */}
      {tab === "skills" && (
        <>
          {/* Install section */}
          <NeonCard variant="cyan">
            <div className="space-y-3">
              <h3 className="text-sm font-bold uppercase tracking-widest text-white/80">
                Install Skill
              </h3>
              <div className="flex gap-2">
                <div className="flex-1">
                  <NeonInput
                    value={installRepo}
                    onChange={setInstallRepo}
                    placeholder="owner/repo/skill-name (GitHub)"
                  />
                </div>
                <NeonButton
                  onClick={handleInstallSkill}
                  disabled={!installRepo.trim() || installing}
                  size="sm"
                >
                  {installing ? "..." : "Install"}
                </NeonButton>
              </div>
              <NeonButton
                onClick={handleInstallBuiltins}
                disabled={installing}
                variant="ghost"
                size="sm"
                className="w-full"
              >
                {installing ? "Installing..." : "Install Builtin Skills"}
              </NeonButton>
            </div>
          </NeonCard>

          {/* Skills list */}
          {skills.length === 0 ? (
            <div className="text-center py-8 text-white/20 text-sm uppercase tracking-widest">
              No skills installed
            </div>
          ) : (
            skills.map((skill) => (
              <NeonCard key={skill.name} variant="cyan">
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <h4 className="text-sm font-bold text-white/80">
                        {skill.name}
                      </h4>
                      <span className="text-[10px] text-neon-cyan/60 bg-neon-cyan/10 px-2 py-0.5 rounded border border-neon-cyan/20 uppercase tracking-widest font-bold">
                        {skill.source}
                      </span>
                    </div>
                    {skill.description && (
                      <p className="text-xs text-white/30 mt-1">
                        {skill.description}
                      </p>
                    )}
                  </div>
                  <NeonButton
                    onClick={() => handleRemoveSkill(skill.name)}
                    variant="danger"
                    size="sm"
                  >
                    Remove
                  </NeonButton>
                </div>
              </NeonCard>
            ))
          )}
        </>
      )}

      {/* Continue button (wizard mode) */}
      {onComplete && (
        <div className="pt-4 space-y-2">
          <NeonButton onClick={onComplete} size="lg" className="w-full">
            Continue to Dashboard →
          </NeonButton>
          <button
            onClick={onComplete}
            className="block mx-auto text-xs text-white/25 hover:text-neon-pink/60 transition-colors uppercase tracking-widest"
          >
            Skip for now
          </button>
        </div>
      )}
    </div>
  );
}
