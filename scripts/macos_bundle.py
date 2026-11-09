"""Small bundle-inspection primitives shared by Mac release tooling."""
from pathlib import Path


MACH_O = {bytes.fromhex(value) for value in (
    'feedface', 'cefaedfe', 'feedfacf', 'cffaedfe',
    'cafebabe', 'bebafeca', 'cafebabf', 'bfbafeca',
)}


def signing_targets(app):
    """Sign nested code before its containing bundle; never follow external links."""
    app = Path(app).resolve(strict=True)
    targets = set()
    for path in app.rglob('*'):
        resolved = path.resolve(strict=True)
        if not resolved.is_relative_to(app):
            raise ValueError('Bundle contains a link outside the app: ' + str(path.relative_to(app)))
        if path.is_symlink():
            continue
        if path.is_file():
            with path.open('rb') as source:
                if source.read(4) in MACH_O:
                    targets.add(path)
        elif path.suffix in ('.framework', '.app', '.xpc', '.bundle'):
            targets.add(path)
    return sorted(targets, key=lambda path: (-len(path.parts), str(path))) + [app]
