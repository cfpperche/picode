param(
    [ValidateSet('inspect', 'install', 'repair')][string]$Operation,
    [string]$Name,
    [string]$ExecutablePath
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)

function ErrorCode($failure) {
    $code = $failure.HResult
    for ($cause = $failure; $null -ne $cause; $cause = $cause.InnerException) {
        $code = $cause.HResult
        if ($code -eq -2147024891 -or $code -eq -2147024894) { break }
    }
    return $code
}

function UserSid([string]$account) {
    if ($account.StartsWith('S-1-')) { return $account }
    return ([System.Security.Principal.NTAccount]::new($account)).Translate(
        [System.Security.Principal.SecurityIdentifier]).Value
}

function FindTask($folder, [string]$taskName) {
    try { return $folder.GetTask($taskName) }
    catch {
        if ((ErrorCode $_.Exception) -eq -2147024894) { return $null }
        throw
    }
}

function HasLogon($definition) {
    foreach ($trigger in $definition.Triggers) {
        if ($trigger.Type -eq 9 -and $trigger.Enabled -and
            ([string]::IsNullOrEmpty($trigger.UserId) -or
             (UserSid $trigger.UserId) -eq (UserSid $definition.Principal.UserId))) { return $true }
    }
    return $false
}

function TriggerLimited($definition) {
    foreach ($trigger in $definition.Triggers) {
        if ($trigger.Enabled -and $trigger.ExecutionTimeLimit -and $trigger.ExecutionTimeLimit -ne 'PT0S') { return $true }
    }
    return $false
}

function ApplyPolicy($settings) {
    $settings.ExecutionTimeLimit = 'PT0S'
    $settings.DisallowStartIfOnBatteries = $false
    $settings.StopIfGoingOnBatteries = $false
    $settings.RunOnlyIfIdle = $false
    $settings.IdleSettings.StopOnIdleEnd = $false
    $settings.RunOnlyIfNetworkAvailable = $false
    $settings.MultipleInstances = 2
    $settings.RestartCount = 3
    $settings.RestartInterval = 'PT1M'
}

function Snapshot($task, [string]$currentSid, [string]$backup) {
    if ($null -eq $task) { return @{ schema = 1; exists = $false } }
    $definition = $task.Definition
    $settings = $definition.Settings
    $action = $null
    if ($definition.Actions.Count -eq 1 -and $definition.Actions.Item(1).Type -eq 0) {
        $action = $definition.Actions.Item(1)
    }
    $path = ''
    $arguments = ''
    if ($null -ne $action) {
        $path = $action.Path.Trim('"')
        $arguments = $action.Arguments
    }
    return [ordered]@{
        schema = 1
        exists = $true
        enabled = [bool]$settings.Enabled
        state = [int]$task.State
        userId = (UserSid $definition.Principal.UserId)
        currentUserId = $currentSid
        interactive = ($definition.Principal.LogonType -eq 3)
        limited = ($definition.Principal.RunLevel -eq 0)
        logon = (HasLogon $definition)
        triggerLimited = (TriggerLimited $definition)
        executable = $path
        arguments = $arguments
        executableExists = ($path -ne '' -and (Test-Path -LiteralPath $path -PathType Leaf))
        executionTimeLimit = [string]$settings.ExecutionTimeLimit
        disallowStartIfOnBatteries = [bool]$settings.DisallowStartIfOnBatteries
        stopIfGoingOnBatteries = [bool]$settings.StopIfGoingOnBatteries
        runOnlyIfIdle = [bool]$settings.RunOnlyIfIdle
        stopOnIdleEnd = [bool]$settings.IdleSettings.StopOnIdleEnd
        runOnlyIfNetworkAvailable = [bool]$settings.RunOnlyIfNetworkAvailable
        multipleInstances = [int]$settings.MultipleInstances
        restartCount = [int]$settings.RestartCount
        restartInterval = [string]$settings.RestartInterval
        lastRun = $task.LastRunTime.ToString('s')
        lastResult = [long]$task.LastTaskResult
        backup = $backup
    }
}

try {
    if ([string]::IsNullOrWhiteSpace($Name) -or $Name.IndexOfAny([IO.Path]::GetInvalidFileNameChars()) -ge 0) {
        throw 'Invalid task name.'
    }
    $service = New-Object -ComObject 'Schedule.Service'
    $service.Connect()
    $folder = $service.GetFolder('\')
    $currentSid = [System.Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    $task = FindTask $folder $Name
    $backup = ''

    if ($Operation -ne 'inspect') {
        if ($null -ne $task -and (UserSid $task.Definition.Principal.UserId) -ne $currentSid) {
            throw 'The startup task belongs to another Windows account; no changes made.'
        }
        if ($Operation -eq 'repair') {
            if ($null -eq $task) { throw 'Startup task is missing. Run picode-desktop install.' }
            $definition = $task.Definition
            if ($definition.Principal.LogonType -ne 3 -or $definition.Principal.RunLevel -ne 0 -or
                -not (HasLogon $definition) -or $definition.Actions.Count -ne 1 -or
                $definition.Actions.Item(1).Type -ne 0 -or $definition.Actions.Item(1).Arguments -ne '--tray' -or
                (TriggerLimited $definition) -or
                -not (Test-Path -LiteralPath ($definition.Actions.Item(1).Path.Trim('"')) -PathType Leaf)) {
                throw 'Startup task is not the expected limited logon task; inspect it before reinstalling.'
            }
        } else {
            if (-not [IO.Path]::IsPathRooted($ExecutablePath) -or -not (Test-Path -LiteralPath $ExecutablePath -PathType Leaf)) {
                throw 'The tray executable must be an existing absolute file path.'
            }
            $definition = $service.NewTask(0)
            $definition.RegistrationInfo.Description = 'PiCode Desktop: start at sign-in with bounded launch retries.'
            $definition.Principal.UserId = $currentSid
            $definition.Principal.LogonType = 3
            $definition.Principal.RunLevel = 0
            $trigger = $definition.Triggers.Create(9)
            $trigger.UserId = $currentSid
            $trigger.Enabled = $true
            $trigger.ExecutionTimeLimit = 'PT0S'
            $action = $definition.Actions.Create(0)
            $action.Path = $ExecutablePath
            $action.Arguments = '--tray'
            $definition.Settings.Enabled = $true
        }
        ApplyPolicy $definition.Settings

        # Preserve a recoverable definition before replacing any existing task.
        if ($null -ne $task) {
            $backupDir = Join-Path ([Environment]::GetFolderPath('LocalApplicationData')) 'PiCode\task-backups'
            [IO.Directory]::CreateDirectory($backupDir) | Out-Null
            $backup = Join-Path $backupDir ($Name + '-' + [Guid]::NewGuid().ToString('N') + '.xml')
            [IO.File]::WriteAllText($backup, $task.Xml, [Text.Encoding]::Unicode)
        }
        $flags = 6 # TASK_CREATE_OR_UPDATE; repair must never recreate a task removed concurrently.
        if ($Operation -eq 'repair') { $flags = 4 }
        $folder.RegisterTaskDefinition($Name, $definition, $flags, $currentSid, $null, 3, $null) | Out-Null
        $task = $folder.GetTask($Name)
    }
    Snapshot $task $currentSid $backup | ConvertTo-Json -Compress -Depth 5
} catch {
    @{ schema = 1; error = $_.Exception.Message; code = (ErrorCode $_.Exception); backup = $backup } |
        ConvertTo-Json -Compress
    exit 1
}
