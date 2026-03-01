; AgentX Inno Setup Installer Script
; Produces AgentX-Setup.exe — double-click to install on Windows

#define MyAppName "AgentX"
#define MyAppPublisher "AgentX Network"
#define MyAppURL "https://agentx.network"
#define MyAppExeName "agentx.exe"

[Setup]
AppId={{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL=https://github.com/Agentx-network/agentx/issues
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
AllowNoIcons=yes
OutputBaseFilename=AgentX-Setup-{#MyAppVersion}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ChangesEnvironment=yes
UninstallDisplayIcon={app}\{#MyAppExeName}
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
CloseApplications=force
CloseApplicationsFilter=*.exe

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "agentx.exe"; DestDir: "{app}"; Flags: ignoreversion restartreplace

[Tasks]
Name: "desktopicon"; Description: "Create a &desktop shortcut"; GroupDescription: "Additional shortcuts:"

[Icons]
Name: "{group}\AgentX Onboard"; Filename: "{app}\{#MyAppExeName}"; Parameters: "onboard"
Name: "{group}\AgentX Chat"; Filename: "{app}\{#MyAppExeName}"; Parameters: "agent"
Name: "{group}\Uninstall AgentX"; Filename: "{uninstallexe}"
Name: "{autodesktop}\AgentX"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; \
  ValueData: "{olddata};{app}"; Check: NeedsAddPath(ExpandConstant('{app}'))

[Run]
Filename: "{app}\{#MyAppExeName}"; Parameters: "onboard"; \
  Description: "Launch AgentX setup wizard"; Flags: nowait postinstall skipifsilent

[Code]
function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  { 1. Remove scheduled task FIRST so it cannot restart the process }
  Exec('schtasks', '/Delete /TN AgentXGateway /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  { 2. Kill running processes }
  Exec('taskkill', '/IM agentx.exe /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec('taskkill', '/IM agentx-desktop.exe /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  { 3. Wait for Windows to release file handles }
  Sleep(2000);
  Result := '';
end;

function NeedsAddPath(Param: string): Boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER,
    'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Uppercase(Param) + ';',
    ';' + Uppercase(OrigPath) + ';') = 0;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Path: string;
  AppDir: string;
  AgentXDir: string;
  ResultCode: Integer;
  P: Integer;
begin
  if CurUninstallStep = usUninstall then
  begin
    AppDir := ExpandConstant('{app}');

    { Run agentx uninstall to clean up gateway service, data, and desktop app }
    Exec(AppDir + '\agentx.exe', 'uninstall --yes', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);

    { Kill any running agentx processes }
    Exec('taskkill', '/IM agentx.exe /F', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);

    { Remove data directory }
    AgentXDir := ExpandConstant('{%USERPROFILE}\.agentx');
    DelTree(AgentXDir, True, True, True);

    { Remove desktop shortcut }
    DeleteFile(ExpandConstant('{autodesktop}\AgentX.lnk'));
  end;

  if CurUninstallStep = usPostUninstall then
  begin
    AppDir := ExpandConstant('{app}');
    if RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path) then
    begin
      P := Pos(';' + Uppercase(AppDir), ';' + Uppercase(Path));
      if P > 0 then
      begin
        Delete(Path, P - 1, Length(AppDir) + 1);
        RegWriteStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', Path);
      end;
    end;
  end;
end;
