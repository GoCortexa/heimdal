; Heimdal Desktop Windows Installer
; NSIS Script for creating Windows installer with Npcap bundling

;--------------------------------
; Includes

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "x64.nsh"

;--------------------------------
; General Configuration

; Application name and version
!define PRODUCT_NAME "Heimdal Desktop"
!define PRODUCT_VERSION "1.0.0"
!define PRODUCT_PUBLISHER "Heimdal Security"
!define PRODUCT_WEB_SITE "https://heimdal.io"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define PRODUCT_UNINST_ROOT_KEY "HKLM"

; Installer name and output file
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "heimdal-desktop-installer-${PRODUCT_VERSION}.exe"

; Default installation directory
InstallDir "$PROGRAMFILES64\Heimdal"

; Request administrator privileges
RequestExecutionLevel admin

; Compression
SetCompressor /SOLID lzma

;--------------------------------
; Interface Settings

!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "${NSISDIR}\Contrib\Graphics\Header\nsis.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "${NSISDIR}\Contrib\Graphics\Wizard\win.bmp"

;--------------------------------
; Pages

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\LICENSE.txt"
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\heimdal-desktop.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Launch Heimdal Desktop"
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

;--------------------------------
; Languages

!insertmacro MUI_LANGUAGE "English"

;--------------------------------
; Version Information

VIProductVersion "1.0.0.0"
VIAddVersionKey "ProductName" "${PRODUCT_NAME}"
VIAddVersionKey "CompanyName" "${PRODUCT_PUBLISHER}"
VIAddVersionKey "LegalCopyright" "Copyright (c) 2024 ${PRODUCT_PUBLISHER}"
VIAddVersionKey "FileDescription" "${PRODUCT_NAME} Installer"
VIAddVersionKey "FileVersion" "${PRODUCT_VERSION}"
VIAddVersionKey "ProductVersion" "${PRODUCT_VERSION}"

;--------------------------------
; Installer Sections

Section "Heimdal Desktop (required)" SecMain
  SectionIn RO
  
  ; Set output path to installation directory
  SetOutPath "$INSTDIR"
  
  ; Copy main executable
  File "..\..\..\bin\heimdal-desktop-windows.exe"
  Rename "$INSTDIR\heimdal-desktop-windows.exe" "$INSTDIR\heimdal-desktop.exe"
  
  ; Copy web dashboard files
  SetOutPath "$INSTDIR\web\dashboard"
  File /r "..\..\..\web\dashboard\*.*"
  
  ; Create default configuration directory
  CreateDirectory "$APPDATA\Heimdal"
  
  ; Copy default configuration if it doesn't exist
  SetOutPath "$APPDATA\Heimdal"
  IfFileExists "$APPDATA\Heimdal\config.json" +2 0
  File "..\..\..\config\config.json"
  
  ; Create database directory
  CreateDirectory "$APPDATA\Heimdal\db"
  
  ; Create logs directory
  CreateDirectory "$LOCALAPPDATA\Heimdal\logs"
  
  ; Write uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"
  
  ; Write registry keys for uninstaller
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\heimdal-desktop.exe"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "NoModify" 1
  WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "NoRepair" 1
  
  ; Create Start Menu shortcuts
  CreateDirectory "$SMPROGRAMS\Heimdal"
  CreateShortCut "$SMPROGRAMS\Heimdal\Heimdal Desktop.lnk" "$INSTDIR\heimdal-desktop.exe"
  CreateShortCut "$SMPROGRAMS\Heimdal\Uninstall.lnk" "$INSTDIR\uninstall.exe"
  
  ; Create Desktop shortcut (optional)
  CreateShortCut "$DESKTOP\Heimdal Desktop.lnk" "$INSTDIR\heimdal-desktop.exe"
  
SectionEnd

Section "Npcap (required for packet capture)" SecNpcap
  SectionIn RO
  
  ; Check if Npcap is already installed
  ReadRegStr $0 HKLM "SOFTWARE\Npcap" "Version"
  ${If} $0 != ""
    DetailPrint "Npcap version $0 is already installed"
    Goto NpcapEnd
  ${EndIf}
  
  ; Extract Npcap installer
  SetOutPath "$TEMP"
  File "npcap-installer.exe"
  
  ; Run Npcap installer silently
  DetailPrint "Installing Npcap..."
  ExecWait '"$TEMP\npcap-installer.exe" /S /loopback_support=yes /winpcap_mode=yes' $0
  
  ${If} $0 != 0
    MessageBox MB_ICONEXCLAMATION "Npcap installation failed. Heimdal Desktop requires Npcap to function properly.$\n$\nPlease install Npcap manually from https://npcap.com/"
  ${Else}
    DetailPrint "Npcap installed successfully"
  ${EndIf}
  
  ; Clean up
  Delete "$TEMP\npcap-installer.exe"
  
  NpcapEnd:
SectionEnd

Section "Windows Service" SecService
  ; Install as Windows Service
  DetailPrint "Installing Heimdal Desktop as Windows Service..."
  
  ; Use the application's built-in service installer
  ExecWait '"$INSTDIR\heimdal-desktop.exe" --install-service' $0
  
  ${If} $0 == 0
    DetailPrint "Service installed successfully"
    
    ; Ask if user wants to start the service now
    MessageBox MB_YESNO "Would you like to start the Heimdal Desktop service now?" IDYES StartService IDNO SkipStart
    
    StartService:
      ExecWait '"$INSTDIR\heimdal-desktop.exe" --start-service' $1
      ${If} $1 == 0
        DetailPrint "Service started successfully"
      ${Else}
        MessageBox MB_ICONEXCLAMATION "Failed to start service. You can start it manually from Services."
      ${EndIf}
    
    SkipStart:
  ${Else}
    MessageBox MB_ICONEXCLAMATION "Failed to install Windows Service. You can run Heimdal Desktop manually."
  ${EndIf}
SectionEnd

Section "Auto-start on boot" SecAutoStart
  ; Configure service to start automatically
  DetailPrint "Configuring auto-start..."
  
  ExecWait '"$INSTDIR\heimdal-desktop.exe" --enable-autostart' $0
  
  ${If} $0 == 0
    DetailPrint "Auto-start configured successfully"
  ${Else}
    DetailPrint "Failed to configure auto-start"
  ${EndIf}
SectionEnd

;--------------------------------
; Section Descriptions

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMain} "Core Heimdal Desktop application files (required)"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecNpcap} "Npcap packet capture driver (required for network monitoring)"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecService} "Install Heimdal Desktop as a Windows Service for background operation"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecAutoStart} "Configure Heimdal Desktop to start automatically when Windows boots"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

;--------------------------------
; Uninstaller Section

Section "Uninstall"
  ; Stop and remove service if installed
  ExecWait '"$INSTDIR\heimdal-desktop.exe" --stop-service'
  ExecWait '"$INSTDIR\heimdal-desktop.exe" --uninstall-service'
  
  ; Remove files
  Delete "$INSTDIR\heimdal-desktop.exe"
  Delete "$INSTDIR\uninstall.exe"
  
  ; Remove web dashboard
  RMDir /r "$INSTDIR\web"
  
  ; Remove installation directory
  RMDir "$INSTDIR"
  
  ; Remove Start Menu shortcuts
  Delete "$SMPROGRAMS\Heimdal\Heimdal Desktop.lnk"
  Delete "$SMPROGRAMS\Heimdal\Uninstall.lnk"
  RMDir "$SMPROGRAMS\Heimdal"
  
  ; Remove Desktop shortcut
  Delete "$DESKTOP\Heimdal Desktop.lnk"
  
  ; Ask if user wants to remove configuration and data
  MessageBox MB_YESNO "Do you want to remove all configuration files and data?$\n$\nThis will delete your network profiles and settings." IDYES RemoveData IDNO KeepData
  
  RemoveData:
    RMDir /r "$APPDATA\Heimdal"
    RMDir /r "$LOCALAPPDATA\Heimdal"
    Goto DataEnd
  
  KeepData:
    DetailPrint "Configuration and data files preserved"
  
  DataEnd:
  
  ; Remove registry keys
  DeleteRegKey ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}"
  
  ; Note: We don't uninstall Npcap as other applications may be using it
  
SectionEnd

;--------------------------------
; Installer Functions

Function .onInit
  ; Check if running on 64-bit Windows
  ${IfNot} ${RunningX64}
    MessageBox MB_ICONSTOP "Heimdal Desktop requires 64-bit Windows."
    Abort
  ${EndIf}
  
  ; Check Windows version (Windows 10 or later)
  ${If} ${AtMostWin8.1}
    MessageBox MB_ICONSTOP "Heimdal Desktop requires Windows 10 or later."
    Abort
  ${EndIf}
  
  ; Check if already installed
  ReadRegStr $0 ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "UninstallString"
  ${If} $0 != ""
    MessageBox MB_YESNO "Heimdal Desktop is already installed. Do you want to uninstall the previous version?" IDYES Uninstall IDNO NoUninstall
    
    Uninstall:
      ExecWait '$0 /S _?=$INSTDIR'
      Goto Continue
    
    NoUninstall:
      Abort
    
    Continue:
  ${EndIf}
FunctionEnd

Function .onInstSuccess
  MessageBox MB_OK "Heimdal Desktop has been successfully installed.$\n$\nPlease ensure you have administrator privileges when running the application for the first time."
FunctionEnd
