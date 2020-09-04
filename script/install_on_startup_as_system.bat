set bin_name=gocroncli.exe
set full_bin_path=%~dp0%bin_name%
set config_dir_name=config
set config_dir_path=%~dp0%config_dir_name%
schtasks /create /sc onstart /ru System /np /f /tn "gocron\startup" /tr "%full_bin_path% \"-config_dir=%config_dir_path%\""
echo "Warning: ExecutionTimeLimit have to cancel it yourself."
echo "By default, the program will be killed after 3 days."
pause
