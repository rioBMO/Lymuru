// Folder name for a downloaded playlist: "<playlist>" or "<playlist> - <owner>".
// Characters that are invalid in paths are removed later by sanitizePath.
export function buildPlaylistFolderName(name: string, owner?: string, includeOwner?: boolean): string {
    const base = name?.trim() || "Playlist";
    const ownerName = owner?.trim();
    return includeOwner && ownerName ? `${base} - ${ownerName}` : base;
}
