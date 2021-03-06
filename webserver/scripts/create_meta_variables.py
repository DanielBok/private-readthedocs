import re
from pathlib import Path
from subprocess import getoutput


def get_git_version():
    version, *_ = getoutput("git tag --sort -version:refname").strip().split("\n")
    if len(version) == 0:
        return "0.0.0"
    return version


def write_version_file():
    fields = {
        "version": get_git_version()
    }

    file = Path(__file__).parent.parent.joinpath("constants.go")
    with open(file, 'r') as f:
        content = original = f.read()

    for key, value in fields.items():
        if isinstance(value, str):
            content = re.sub(fr'{key} = ".*"', f'{key} = "{value}"', content)

    if content != original:
        with open(file, 'w') as f:
            f.write(content)
        print("Generated constants file")
    else:
        print("Similar contents in constants file. Generation skipped")

if __name__ == "__main__":
    write_version_file()
