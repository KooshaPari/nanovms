$root = 'C:\Users\koosh\infra-research\nanovms'
Get-ChildItem -Path $root -Recurse -Include *.go -File | Where-Object { $_.FullName -notmatch 'third_party|\.worktrees|node_modules' } | ForEach-Object {
  $rel = $_.FullName.Substring($root.Length+1)
  $lines = (Get-Content $_.FullName | Measure-Object -Line).Lines
  '{0,-65} {1,6}' -f $rel, $lines
} | Sort-Object