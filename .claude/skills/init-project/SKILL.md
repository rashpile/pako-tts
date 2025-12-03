---
name: init-project
description: Initialize a new project using Specify. Use this skill when the user wants to initialize, bootstrap, or set up a new project with Specify.
allowed-tools: Bash
---

# Initialize Project with Specify

## Instructions

When the user asks to initialize a project or set up a new project with Specify:

1. Run the following command in the current directory:
   ```bash
   specify init .
   ```

2. Wait for the command to complete and report the results to the user.

3. If there are any errors, help the user troubleshoot based on the error message.

## Trigger Conditions

Use this skill when the user mentions:
- "init project"
- "initialize project"
- "bootstrap project"
- "set up project with specify"
- "specify init"
