update-scripts:
	@curl --request PUT -u $(mirrors) --url https://mirrors.tencent.com/repository/generic/cli-market/env/windows/metax.ps1 --upload-file scripts/t2.ps1
	@curl --request PUT -u $(mirrors) --url https://mirrors.tencent.com/repository/generic/cli-market/env/unix-like/metax.sh --upload-file scripts/t2.sh
	@echo "\n\033[32m已成功更新脚本\033[0m"

.PHONY: update-scripts
