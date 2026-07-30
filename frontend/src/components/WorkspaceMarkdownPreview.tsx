import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

export default function WorkspaceMarkdownPreview({ content }: { content: string }) {
  return (
    <article className="rounded-xl border border-slate-200/80 bg-white px-5 py-6 shadow-sm sm:px-8">
      <div
        className="min-w-0 break-words text-[15px] leading-7 text-slate-700
          [&>*:first-child]:mt-0 [&>*:last-child]:mb-0
          [&_h1]:mb-5 [&_h1]:mt-8 [&_h1]:border-b [&_h1]:border-slate-200 [&_h1]:pb-3 [&_h1]:text-2xl [&_h1]:font-bold [&_h1]:tracking-tight [&_h1]:text-slate-950
          [&_h2]:mb-3 [&_h2]:mt-7 [&_h2]:border-b [&_h2]:border-slate-100 [&_h2]:pb-2 [&_h2]:text-xl [&_h2]:font-bold [&_h2]:text-slate-900
          [&_h3]:mb-2 [&_h3]:mt-6 [&_h3]:text-lg [&_h3]:font-semibold [&_h3]:text-slate-900
          [&_h4]:mb-2 [&_h4]:mt-5 [&_h4]:font-semibold [&_h4]:text-slate-900
          [&_p]:my-3
          [&_a]:font-medium [&_a]:text-cyan-700 [&_a]:underline [&_a]:decoration-cyan-300 [&_a]:underline-offset-2 hover:[&_a]:text-cyan-800
          [&_strong]:font-semibold [&_strong]:text-slate-900
          [&_ul]:my-3 [&_ul]:list-disc [&_ul]:space-y-1.5 [&_ul]:pl-6
          [&_ol]:my-3 [&_ol]:list-decimal [&_ol]:space-y-1.5 [&_ol]:pl-6
          [&_li]:pl-1 [&_li>p]:my-1
          [&_input[type=checkbox]]:mr-2 [&_input[type=checkbox]]:accent-cyan-600
          [&_blockquote]:my-4 [&_blockquote]:border-l-4 [&_blockquote]:border-cyan-200 [&_blockquote]:bg-cyan-50/60 [&_blockquote]:px-4 [&_blockquote]:py-2 [&_blockquote]:text-slate-600
          [&_hr]:my-7 [&_hr]:border-slate-200
          [&_code]:rounded-md [&_code]:bg-slate-100 [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[0.88em] [&_code]:text-rose-700
          [&_pre]:my-4 [&_pre]:overflow-x-auto [&_pre]:rounded-xl [&_pre]:border [&_pre]:border-slate-800 [&_pre]:bg-slate-950 [&_pre]:p-4 [&_pre]:shadow-inner
          [&_pre_code]:bg-transparent [&_pre_code]:p-0 [&_pre_code]:text-[13px] [&_pre_code]:leading-6 [&_pre_code]:text-slate-100
          [&_table]:my-5 [&_table]:w-full [&_table]:border-collapse [&_table]:overflow-hidden [&_table]:text-sm
          [&_thead]:bg-slate-100 [&_th]:border [&_th]:border-slate-200 [&_th]:px-3 [&_th]:py-2.5 [&_th]:text-left [&_th]:font-semibold [&_th]:text-slate-900
          [&_td]:border [&_td]:border-slate-200 [&_td]:px-3 [&_td]:py-2.5 [&_td]:align-top
          [&_tbody_tr:nth-child(even)]:bg-slate-50/70
          [&_img]:my-5 [&_img]:max-w-full [&_img]:rounded-xl [&_img]:border [&_img]:border-slate-200"
      >
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          components={{
            a: ({ href, children, ...props }) => {
              const external = /^https?:\/\//i.test(href || "");
              return (
                <a
                  {...props}
                  href={href}
                  target={external ? "_blank" : undefined}
                  rel={external ? "noreferrer noopener" : undefined}
                >
                  {children}
                </a>
              );
            },
          }}
        >
          {content}
        </ReactMarkdown>
      </div>
    </article>
  );
}
