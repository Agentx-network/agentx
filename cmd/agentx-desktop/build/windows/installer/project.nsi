Unicode true

!define INFO_PROJECTNAME    "agentx-desktop"
!define INFO_COMPANYNAME    "AgentX Network"
!define INFO_PRODUCTNAME    "AgentX Desktop"
!define INFO_PRODUCTVERSION "0.8.10"
!define INFO_COPYRIGHT      "Copyright 2026 AgentX Network"

!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Launch AgentX Desktop"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
ShowInstDetails show

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext

    # Remove scheduled task first so it cannot restart the process
    nsExec::ExecToLog 'schtasks /Delete /TN AgentXGateway /F'

    # Kill running processes so installer can overwrite binaries
    nsExec::ExecToLog 'taskkill /IM agentx-desktop.exe /F'
    nsExec::ExecToLog 'taskkill /IM agentx.exe /F'

    # Wait for Windows to release file handles
    Sleep 2000

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    # Desktop shortcut
    CreateShortCut "$DESKTOP\AgentX Desktop.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0

    # Start Menu shortcuts
    CreateDirectory "$SMPROGRAMS\AgentX"
    CreateShortCut "$SMPROGRAMS\AgentX\AgentX Desktop.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\${PRODUCT_EXECUTABLE}" 0
    CreateShortCut "$SMPROGRAMS\AgentX\Uninstall AgentX Desktop.lnk" "$INSTDIR\uninstall.exe"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    # Kill running processes before removal
    nsExec::ExecToLog 'taskkill /IM agentx-desktop.exe /F'
    nsExec::ExecToLog 'taskkill /IM agentx.exe /F'

    # Remove AgentXGateway scheduled task
    nsExec::ExecToLog 'schtasks /Delete /TN AgentXGateway /F'

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"

    # Remove AgentX data directory
    RMDir /r "$PROFILE\.agentx"

    RMDir /r $INSTDIR

    # Remove shortcuts
    Delete "$DESKTOP\AgentX Desktop.lnk"
    Delete "$SMPROGRAMS\AgentX\AgentX Desktop.lnk"
    Delete "$SMPROGRAMS\AgentX\Uninstall AgentX Desktop.lnk"
    RMDir "$SMPROGRAMS\AgentX"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
