// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import AppKit
import SwiftUI

struct AttachmentRootsView: View {
    @ObservedObject var store: LedgerStore
    @Environment(\.dismiss) private var dismiss
    @StateObject private var chooser = AttachmentDirectoryChooser()
    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("附件目录 / Attachment directories").font(.headline)
            Text("默认目录为账本入口所在目录。额外目录只通过显式选择授权；撤销后旧引用不可打开。 / Additional directories require explicit selection; removing one invalidates previous references.")
                .font(.callout)
            if let roots = store.attachmentRoots {
                Text("默认（不可移除） / Default: " + roots.default_root).textSelection(.enabled)
                ScrollView {
                  VStack(alignment: .leading) {
                    ForEach(roots.additional_roots, id: \.self) { root in
                        HStack {
                            Text(root).textSelection(.enabled)
                            Spacer()
                            Button("移除 / Remove") { Task { await store.updateAttachmentRoots(roots.additional_roots.filter { $0 != root }, expectedRevision: roots.policy_revision) } }
                                .disabled(store.attachmentRootsBusy || chooser.choosing)
                        }
                    }
                  }
                }.frame(minHeight: 120)
                Button("刷新附件 / Refresh attachments") { Task { await store.updateAttachmentRoots(roots.additional_roots, persist: false) } }.disabled(store.attachmentRootsBusy || chooser.choosing)
                Button("添加目录… / Add directory…", action: chooseDirectory)
                    .disabled(store.attachmentRootsBusy || chooser.choosing || roots.additional_roots.count >= 32)
            } else { Text("目录配置暂不可用 / Directory configuration unavailable") }
            if let notice = store.attachmentRootNotice { Text(notice).foregroundStyle(.orange).textSelection(.enabled) }
            if store.attachmentRootsBusy { ProgressView() }
            HStack { Spacer(); Button("完成 / Done") { dismiss() }.keyboardShortcut(.defaultAction) }
        }.padding(20).frame(width: 650)
        .onDisappear { chooser.cancel() }
    }
    private func chooseDirectory() { chooser.choose(store: store) }
}

@MainActor
final class AttachmentDirectoryChooser: ObservableObject {
    @Published var choosing = false
    private var panel: NSOpenPanel?
    func cancel() { panel?.cancel(nil); panel = nil; choosing = false }
    func choose(store: LedgerStore) {
        guard !choosing, let roots = store.attachmentRoots, let window = NSApp.keyWindow else { return }
        let chooser = NSOpenPanel()
        chooser.canChooseDirectories = true
        chooser.canChooseFiles = false
        chooser.allowsMultipleSelection = false
        chooser.canCreateDirectories = false
        chooser.resolvesAliases = false
        chooser.message = "选择允许打开账本附件的目录 / Select a directory containing ledger attachments"
        choosing = true
        panel = chooser
        chooser.beginSheetModal(for: window) { [weak self, weak store] response in
            Task { @MainActor in
                guard let self else { return }
                self.choosing = false; self.panel = nil
                guard response == .OK, let url = chooser.url, let store else { return }
                let selected = url.standardizedFileURL.resolvingSymlinksInPath().path
                if selected == "/" || selected == FileManager.default.homeDirectoryForCurrentUser.resolvingSymlinksInPath().path || selected == "/Users" || selected == "/Volumes" {
                    let alert = NSAlert()
                    alert.messageText = "允许访问整个所选目录？ / Allow the entire selected directory?"
                    alert.informativeText = "范围：" + selected + "\n该目录及所有子目录中的账本附件都将可打开。 / Ledger attachments anywhere under this directory can be opened."
                    alert.addButton(withTitle: "允许 / Allow")
                    alert.addButton(withTitle: "取消 / Cancel")
                    alert.beginSheetModal(for: window) { response in
                        guard response == .alertFirstButtonReturn else { return }
                        Task { @MainActor in await store.updateAttachmentRoots(roots.additional_roots + [selected], expectedRevision: roots.policy_revision) }
                    }
                    return
                }
                await store.updateAttachmentRoots(roots.additional_roots + [url.path], expectedRevision: roots.policy_revision)
            }
        }
    }
}
