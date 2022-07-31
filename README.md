# Keeps

Keeps is a terminal ui for keepass. It currently has very minimal functionality.
It allows you to open a kdbx file, copy entries (from the first group only),
create new entries, and save back the files.

I built keeps because the original keepass client doesn't look good on my machine,
some keyboard shortcuts don't work in i3, and I felt inspired by [Bubble Tea](https://github.com/charmbracelet/bubbletea).

**IMPORTANT**: Keep a backup of your database file. During development I corrupted my file once,
and I don't know why, so better be careful.

## Installation

You can install keeps using `go install`

```bash
go install github.com/flatplate/keeps@latest
```

## Usage

Pass the database path as an argument to keeps.

```bash
$ keeps /path/to/database.kdbx
```

After starting it will prompt you for your password. Type in your password and press
enter to continue to the table view where the entries are listed.

Here you can use the following keybindings:

- `j/k`: move down/up
- `y`: copy the currently highlighted password
- `o`: create new entry
- `w`: save database
- `ctrl+c`: exit

In new entry creation view you can use:

- `tab/shift+tab`: move to next/previous input
- `enter`: save entry
- `esc`: cancel and go back to table view

## Todo

- Filter entries based on title / url whatever
- Use multiple groups

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License
[MIT](https://choosealicense.com/licenses/mit/)
