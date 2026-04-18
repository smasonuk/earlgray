import re

with open("tui_test.go", "r") as f:
    lines = f.readlines()

new_lines = []
for i, line in enumerate(lines):
    new_lines.append(line)
    if "rt.HandleEvent" in line and not "consumed :=" in line and "Test" in "".join(lines[:i]):
        # Add rt.Update(root) if next line doesn't have it
        if i + 1 < len(lines) and "if rt.IsDirty()" not in lines[i+1]:
            new_lines.append("\tif rt.IsDirty() { rt.Update(root) }\n")

with open("tui_test.go", "w") as f:
    f.writelines(new_lines)
