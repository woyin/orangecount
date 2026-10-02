// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
import Foundation
import Darwin

enum AnalysisFileExportError: Error { case invalidCapture, protectedTarget, unsupportedTarget, filesystem(Int32) }

// A fixed byte capture and retained protected inode identities. Save never
// reads query state; a directory descriptor anchors temporary creation/rename.
struct AnalysisFileExport: Sendable {
    private struct Identity: Hashable, Sendable {
        let device: Int32
        let inode: UInt64
        init(_ info: stat) { device = info.st_dev; inode = info.st_ino }
    }
    private let content: Data
    private let protectedPaths: [String]
    private let identities: Set<Identity>

    init(content: Data, protectedPaths: [String]) throws {
        guard content.count <= 16 * 1024 * 1024, !protectedPaths.isEmpty,
              protectedPaths.allSatisfy({ $0.hasPrefix("/") }) else { throw AnalysisFileExportError.invalidCapture }
        self.content = content
        self.protectedPaths = protectedPaths
        var retained = Set<Identity>()
        for path in protectedPaths {
            if let info = try Self.info(path) { retained.insert(Identity(info)) }
        }
        identities = retained
    }

    private init(content: Data, protectedPaths: [String], retainedIdentities: Set<Identity>) {
        self.content = content
        self.protectedPaths = protectedPaths
        identities = retainedIdentities
    }

    func addingProtectedPaths(_ paths: [String]) throws -> AnalysisFileExport {
        let refreshed = try AnalysisFileExport(content: content, protectedPaths: Array(Set(protectedPaths + paths)).sorted())
        return AnalysisFileExport(content: content, protectedPaths: refreshed.protectedPaths,
            retainedIdentities: identities.union(refreshed.identities))
    }

    func save(to destination: URL) throws {
        try Task.checkCancellation()
        guard destination.isFileURL else { throw AnalysisFileExportError.unsupportedTarget }
        let parent = destination.deletingLastPathComponent().standardizedFileURL.resolvingSymlinksInPath()
        let name = destination.lastPathComponent
        guard !name.isEmpty, name != ".", name != ".." else { throw AnalysisFileExportError.unsupportedTarget }
        let target = parent.appendingPathComponent(name)
        try validate(target)
        let directory = Darwin.open(parent.path, O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW)
        guard directory >= 0 else { throw AnalysisFileExportError.filesystem(errno) }
        defer { Darwin.close(directory) }
        var directoryInfo = stat()
        guard fstat(directory, &directoryInfo) == 0 else { throw AnalysisFileExportError.filesystem(errno) }
        let temporary = ".orangecount-export-\(UUID().uuidString)"
        let descriptor = openat(directory, temporary, O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, mode_t(0o600))
        guard descriptor >= 0 else { throw AnalysisFileExportError.filesystem(errno) }
        var opened = true
        defer {
            if opened { Darwin.close(descriptor) }
            unlinkat(directory, temporary, 0)
        }
        try content.withUnsafeBytes { buffer in
            var offset = 0
            while offset < buffer.count {
                let count = Darwin.write(descriptor, buffer.baseAddress!.advanced(by: offset), buffer.count - offset)
                if count < 0 && errno == EINTR { continue }
                guard count > 0 else { throw AnalysisFileExportError.filesystem(errno) }
                offset += count
            }
        }
        guard fsync(descriptor) == 0 else { throw AnalysisFileExportError.filesystem(errno) }
        opened = false
        guard Darwin.close(descriptor) == 0 else { throw AnalysisFileExportError.filesystem(errno) }
        guard let currentDirectory = try Self.info(parent.path), Identity(currentDirectory) == Identity(directoryInfo) else {
            throw AnalysisFileExportError.unsupportedTarget
        }
        guard destination.deletingLastPathComponent().standardizedFileURL.resolvingSymlinksInPath().path == parent.path else {
            throw AnalysisFileExportError.unsupportedTarget
        }
        try validate(target)
        try Task.checkCancellation()
        guard renameat(directory, temporary, directory, name) == 0 else { throw AnalysisFileExportError.filesystem(errno) }
    }

    private func validate(_ target: URL) throws {
        var lexicalInfo = stat()
        if lstat(target.path, &lexicalInfo) == 0 {
            guard (lexicalInfo.st_mode & S_IFMT) == S_IFREG else { throw AnalysisFileExportError.unsupportedTarget }
            if (try target.resourceValues(forKeys: [.isAliasFileKey])).isAliasFile == true { throw AnalysisFileExportError.unsupportedTarget }
        } else if errno != ENOENT { throw AnalysisFileExportError.filesystem(errno) }
        let targetPath = target.standardizedFileURL.resolvingSymlinksInPath().path
        let targetInfo = try Self.info(targetPath)
        if let targetInfo, identities.contains(Identity(targetInfo)) { throw AnalysisFileExportError.protectedTarget }
        for path in protectedPaths {
            let protected = URL(fileURLWithPath: path).standardizedFileURL.resolvingSymlinksInPath().path
            if protected == targetPath { throw AnalysisFileExportError.protectedTarget }
            if let current = try Self.info(protected), let targetInfo, Identity(current) == Identity(targetInfo) {
                throw AnalysisFileExportError.protectedTarget
            }
        }
    }

    private static func info(_ path: String) throws -> stat? {
        var info = stat()
        if fstatat(AT_FDCWD, path, &info, 0) == 0 { return info }
        if errno == ENOENT || errno == ENOTDIR { return nil }
        throw AnalysisFileExportError.filesystem(errno)
    }
}
