// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Foundation
import Darwin

// Bookmark storage is deliberately separate from accounting and views. The
// current ad hoc application is not sandboxed: these are ordinary bookmarks,
// not an assertion that macOS has granted security-scoped access.
@MainActor
final class AttachmentRootPreferences {
    struct Directory: Codable, Sendable {
        let path: String
        let bookmark: Data
        let device: UInt64
        let inode: UInt64
    }
    struct Configuration: Codable, Sendable {
        let generation: UInt64
        let directories: [Directory]
    }
    struct Restoration: Sendable {
        let generation: UInt64
        let roots: [String]
        let invalid: [String]
    }
    static let changed = Notification.Name("org.orangecount.attachmentRootsChanged")
    private let defaults: UserDefaults?
    init(defaults: UserDefaults?) { self.defaults = defaults }
    static func ledgerIdentity(_ path: String) -> String {
        URL(fileURLWithPath: path).standardizedFileURL.resolvingSymlinksInPath().path
    }
    private func key(_ ledger: String) -> String { "org.orangecount.attachmentRoots." + Self.ledgerIdentity(ledger) }
    private func configuration(_ ledger: String) throws -> Configuration {
        guard let data = defaults?.data(forKey: key(ledger)) else { return Configuration(generation: 0, directories: []) }
        return try JSONDecoder().decode(Configuration.self, from: data)
    }
    private static func identity(_ url: URL) throws -> (UInt64, UInt64) {
        var info = stat()
        guard lstat(url.path, &info) == 0, (info.st_mode & S_IFMT) == S_IFDIR else { throw CocoaError(.fileReadUnknown) }
        return (UInt64(info.st_dev), UInt64(info.st_ino))
    }
    func restore(_ ledger: String) throws -> Restoration {
        let config = try configuration(ledger)
        var roots: [String] = [], invalid: [String] = []
        for directory in config.directories {
            do {
                var stale = false
                let url = try URL(resolvingBookmarkData: directory.bookmark, options: [.withoutUI, .withoutMounting], relativeTo: nil, bookmarkDataIsStale: &stale).resolvingSymlinksInPath()
                let identity = try Self.identity(url)
                guard !stale, url.path == directory.path, identity.0 == directory.device, identity.1 == directory.inode else { throw CocoaError(.fileReadUnknown) }
                roots.append(url.path)
            } catch { invalid.append(directory.path) }
        }
        return Restoration(generation: config.generation, roots: roots, invalid: invalid)
    }
    // Prepare bookmarks before modifying the active Go policy. A later save
    // failure is surfaced as session-only configuration, never silent success.
    func prepare(_ roots: [String], ledger: String) throws -> Configuration {
        let old = try configuration(ledger)
        guard old.generation < UInt64.max, roots.count <= 32 else { throw CocoaError(.fileWriteUnknown) }
        var seen: Set<String> = [], directories: [Directory] = []
        for path in roots {
            let url = URL(fileURLWithPath: path).standardizedFileURL.resolvingSymlinksInPath()
            guard seen.insert(url.path).inserted else { continue }
            let identity = try Self.identity(url)
            let bookmark = try url.bookmarkData(options: [], includingResourceValuesForKeys: nil, relativeTo: nil)
            directories.append(Directory(path: url.path, bookmark: bookmark, device: identity.0, inode: identity.1))
        }
        return Configuration(generation: old.generation + 1, directories: directories)
    }
    func save(_ configuration: Configuration, ledger: String) throws {
        guard let defaults else { return }
        guard try self.configuration(ledger).generation < configuration.generation else { throw CocoaError(.fileWriteUnknown) }
        let data = try JSONEncoder().encode(configuration)
        defaults.set(data, forKey: key(ledger))
        guard defaults.data(forKey: key(ledger)) == data else { throw CocoaError(.fileWriteUnknown) }
        NotificationCenter.default.post(name: Self.changed, object: Self.ledgerIdentity(ledger))
    }
}
