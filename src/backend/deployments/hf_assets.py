import os
import argparse

from huggingface_hub import snapshot_download

parser = argparse.ArgumentParser(
    prog="hf_assets",
    description="Helper script for downloading the necessary assets from the Hugging Face Hub REPO.",
)

parser.add_argument(
    "-t",
    "--token",
    type=str,
    help="Hugging Face Hub token for authentication. If not provided, the script will use the token from the environment variable",
)

parser.add_argument(
    "-r",
    "--repo-id",
    type=str,
    help="Hugging Face Hub repository ID",
)

if __name__ == "__main__":
    args = parser.parse_args()
    token, repo_id = args.token, args.repo_id

    if not token:
        token = os.getenv("HF_TOKEN")

    if not token:
        raise ValueError(
            "Hugging Face Hub token is required. Please provide it via --token or set the HF_TOKEN environment variable."
        )

    if not repo_id:
        raise ValueError(
            "Hugging Face Hub repository ID is required. Please provide it via --repo-id."
        )

    snapshot_download(
        repo_id=repo_id,
        token=token,
        local_dir="configs",
        revision="main",
    )
