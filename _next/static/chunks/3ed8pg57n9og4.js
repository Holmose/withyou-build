(globalThis.TURBOPACK||(globalThis.TURBOPACK=[])).push(["object"==typeof document?document.currentScript:void 0,784669,e=>{"use strict";var t=e.i(718374),s=e.i(472069),a=e.i(135467),i=e.i(129259),o=e.i(72925),l=e.i(288371),n=e.i(699783),r=e.i(957264),c=e.i(289601),d=e.i(54914),p=e.i(689734);function m({id:e,label:s,description:a,value:i,fallbackValue:o,dirty:n,disabled:u,modelOptions:h,onChange:x}){let g=(0,l.useTranslations)("common.states"),f=n?(0,t.jsx)(c.Badge,{variant:"ghost",className:"relative -mt-1.5 text-[8px] font-medium text-amber-800",children:g("unsaved")}):null;return(0,t.jsx)(r.motion.div,{layout:!0,transition:{duration:.2,ease:[.22,1,.36,1]},children:(0,t.jsx)(d.Field,{children:(0,t.jsxs)("div",{className:"flex flex-col gap-2 md:flex-row md:items-start md:gap-4 xl:gap-6",children:[(0,t.jsxs)("div",{className:"min-w-0 flex-1",children:[(0,t.jsxs)("div",{className:"flex items-center gap-2",children:[(0,t.jsx)(d.FieldLabel,{htmlFor:e,children:s}),f]}),a?(0,t.jsx)(d.FieldDescription,{className:"text-[11px]",children:a}):null]}),(0,t.jsx)("div",{className:"w-full min-w-0 md:w-44 md:shrink-0 xl:w-52",children:(0,t.jsx)(p.ModelSelect,{id:e,value:i,fallbackValue:o,disabled:u,options:h,onChange:x})})]})})})}var u=e.i(293646),h=e.i(655096);let x=(0,e.i(146009).default)("file-box",[["path",{d:"M14.5 22H18a2 2 0 0 0 2-2V8a2.4 2.4 0 0 0-.706-1.706l-3.588-3.588A2.4 2.4 0 0 0 14 2H6a2 2 0 0 0-2 2v3.8",key:"1kchwa"}],["path",{d:"M14 2v5a1 1 0 0 0 1 1h5",key:"wfsgrz"}],["path",{d:"M11.7 14.2 7 17l-4.7-2.8",key:"1yk8tc"}],["path",{d:"M3 13.1a2 2 0 0 0-.999 1.76v3.24a2 2 0 0 0 .969 1.78L6 21.7a2 2 0 0 0 2.03.01L11 19.9a2 2 0 0 0 1-1.76V14.9a2 2 0 0 0-.97-1.78L8 11.3a2 2 0 0 0-2.03-.01z",key:"19flxy"}],["path",{d:"M7 17v5",key:"1yj1jh"}]]);var g=e.i(536411),f=e.i(570434),_=e.i(651159),y=e.i(328549),v=e.i(495823),b=e.i(740011),k=e.i(599796),j=e.i(351861),w=e.i(947836),C=e.i(781349),S=e.i(84380),T=e.i(691456),N=e.i(1326),P=e.i(360887),D=e.i(532964),M=e.i(435945),A=e.i(102025),L=e.i(141843),E=e.i(991132),O=e.i(732671),I=e.i(145622),F=e.i(76251);let R={name:"",description:"",content:"",enabled:!0};var z=e.i(373514),B=e.i(588491),H=e.i(845526);function W({action:e,dirty:a,disabled:i,onChange:o,value:n}){let r=(0,l.useTranslations)("adminPrompts"),c=s.useMemo(()=>({id:"chat.skills_prompt",label:r("settings.skillsPrompt.label"),description:r("settings.skillsPrompt.description"),type:"textarea",placeholder:r("settings.defaultPromptPlaceholder")}),[r]);return(0,t.jsxs)("div",{className:"relative",children:[(0,t.jsx)(H.SettingsFieldList,{children:(0,t.jsx)(H.SettingsFieldItem,{children:(0,t.jsx)(u.SettingsFieldEditor,{field:c,value:n,dirty:a,disabled:i,onChange:o})})}),e?(0,t.jsx)("div",{className:"absolute right-0 top-0 z-10",children:e}):null]})}function q({emptyLabel:e,getSummary:s,icon:a,items:i,loading:o,onDelete:n,onEdit:r,onEnabledChange:c}){let d=(0,l.useTranslations)("adminPrompts"),p=(0,_.useLocale)(),m=o&&0===i.length,u=i.length>0,h=(0,P.useVirtualTableRows)(i,{estimateSize:40});return(0,t.jsxs)(S.Table,{viewportRef:h.viewportRef,viewportClassName:h.viewportClassName,viewportStyle:h.viewportStyle,children:[(0,t.jsx)(S.TableHeader,{children:(0,t.jsxs)(S.TableRow,{className:"hover:bg-transparent",children:[(0,t.jsx)(S.TableHead,{className:"w-[220px]",children:d("fields.name")}),(0,t.jsx)(S.TableHead,{className:"w-[320px]",children:d("fields.description")}),(0,t.jsx)(S.TableHead,{className:"w-[96px] text-center",children:d("fields.enabled")}),(0,t.jsx)(S.TableHead,{className:"w-[160px]",children:d("fields.createdAt")}),(0,t.jsx)(S.TableHead,{className:"w-[160px]",children:d("fields.updatedAt")}),(0,t.jsx)(S.TableHead,{className:"w-[56px]",stickyEnd:!0})]})}),(0,t.jsxs)(S.TableBody,{children:[m?(0,t.jsx)(S.TableLoadingRow,{colSpan:6}):null,0!==i.length||o?null:(0,t.jsx)(S.TableEmptyRow,{colSpan:6,children:e}),u?(0,t.jsx)(P.VirtualTablePaddingRow,{colSpan:6,height:h.paddingTop}):null,u?h.rows.map(({item:e})=>{let i=e.trigger||e.title,o=s(e);return(0,t.jsxs)(S.TableRow,{interactive:!0,onClick:()=>r(e),children:[(0,t.jsx)(S.TableCell,{children:(0,t.jsxs)("div",{className:"flex max-w-[200px] min-w-0 items-center gap-2",children:[(0,t.jsx)(a,{className:"size-4 shrink-0 text-muted-foreground",strokeWidth:1.8}),(0,t.jsx)("span",{className:"min-w-0 truncate font-medium",children:i})]})}),(0,t.jsx)(S.TableCell,{children:(0,t.jsx)("p",{className:"max-w-[300px] truncate text-muted-foreground",children:o})}),(0,t.jsx)(S.TableCell,{className:"text-center",children:(0,t.jsx)("div",{className:"flex h-7 items-center justify-center",children:(0,t.jsx)(w.Switch,{size:"sm",checked:e.enabled,onClick:e=>e.stopPropagation(),onCheckedChange:t=>c(e,t),"aria-label":e.enabled?d("disable"):d("enable")})})}),(0,t.jsx)(S.TableCell,{className:"whitespace-nowrap text-muted-foreground",children:(0,z.formatDateTime)(e.createdAt,p)}),(0,t.jsx)(S.TableCell,{className:"whitespace-nowrap text-muted-foreground",children:(0,z.formatDateTime)(e.updatedAt,p)}),(0,t.jsx)(S.TableCell,{stickyEnd:!0,children:(0,t.jsx)(v.Button,{type:"button",variant:"ghost",size:"icon",className:"h-7 w-7 text-muted-foreground shadow-none hover:bg-muted hover:text-destructive",onClick:t=>{t.stopPropagation(),n(e)},"aria-label":d("delete"),children:(0,t.jsx)(f.Trash2,{className:"size-3.5",strokeWidth:1.6})})})]},e.id)}):null,u?(0,t.jsx)(P.VirtualTablePaddingRow,{colSpan:6,height:h.paddingBottom}):null]})]})}function K(){let e=(0,l.useTranslations)("adminPrompts"),a=(0,l.useTranslations)("common"),[i,r]=s.useState("skills"),[c,d]=s.useState(""),[p,m]=s.useState(""),[u,f]=s.useState(!0),[_,S]=s.useState(!1),P=function(){let e=(0,l.useTranslations)("adminPrompts"),{accessToken:t}=(0,A.useAuthSession)(),[a,i]=s.useState([]),[o,r]=s.useState(0),[c,d]=s.useState(1),[p,m]=s.useState(25),[u,h]=s.useState(""),[x,g]=s.useState(""),[f,_]=s.useState(!0),[y,v]=s.useState(!1),[b,k]=s.useState(R),[j,w]=s.useState(!1),[C,S]=s.useState(null),[,T]=s.useTransition(),N=s.useRef(0);s.useEffect(()=>{let e=window.setTimeout(()=>{g(u.trim())},250);return()=>window.clearTimeout(e)},[u]);let P=s.useCallback(async()=>{let s=N.current+1;N.current=s,_(!0);try{let e=await (0,I.listAdminPromptPresets)(t,{page:c,pageSize:p,query:x});if(s!==N.current)return;T(()=>{i(e.results),r(e.total)})}catch(t){n.toast.error(e("toast.loadFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}finally{s===N.current&&_(!1)}},[t,x,c,p,T,e]);s.useEffect(()=>{P()},[P]);let D=Math.max(1,Math.ceil(o/p)),M=s.useCallback(e=>{h(e),d(1)},[]),E=s.useCallback(e=>{m(e),d(1)},[]),z=s.useCallback(()=>{k(R),w(!0)},[]),B=s.useCallback(e=>{k({id:e.id,name:e.trigger||e.title,description:e.description,content:e.content,enabled:e.enabled}),w(!0)},[]),H=s.useCallback(async()=>{let s,a={title:s=(0,F.normalizePromptPresetName)(b.name),trigger:s,description:b.description.trim(),content:b.content.trim(),enabled:b.enabled,sortOrder:0};if(!a.title||!a.trigger||!a.content)return void n.toast.error(e("toast.invalid"));if((0,F.normalizePromptPresetName)(b.name).length>F.PROMPT_PRESET_LIMITS.name||b.description.trim().length>F.PROMPT_PRESET_LIMITS.description||b.content.trim().length>F.PROMPT_PRESET_LIMITS.content)return void n.toast.error(e("toast.tooLong"));v(!0);try{b.id?(await (0,I.updateAdminPromptPreset)(t,b.id,a),await P(),n.toast.success(e("toast.updated"))):(await (0,I.createAdminPromptPreset)(t,a),await P(),n.toast.success(e("toast.created"))),w(!1)}catch(t){n.toast.error(b.id?e("toast.updateFailed"):e("toast.createFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}finally{v(!1)}},[t,b,P,e]),W=s.useCallback(async(s,a)=>{i(e=>e.map(e=>e.id===s.id?{...e,enabled:a}:e));try{await (0,I.updateAdminPromptPreset)(t,s.id,{enabled:a}),await P()}catch(t){i(e=>(0,L.replaceByID)(e,s.id,e=>e.id,s)),n.toast.error(e("toast.updateFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}},[t,P,e]),q=s.useCallback(async()=>{if(C){S(null);try{await (0,I.deleteAdminPromptPreset)(t,C.id),i(e=>(0,L.removeByID)(e,C.id,e=>e.id)),r(e=>Math.max(0,e-1)),await P(),n.toast.success(e("toast.deleted"))}catch(t){n.toast.error(e("toast.deleteFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}}},[t,C,P,e]);return{items:a,total:o,page:c,pageSize:p,pageCount:D,query:u,loading:f,saving:y,form:b,dialogOpen:j,deleteTarget:C,setPage:d,setPageSize:E,setQuery:M,setForm:k,setDialogOpen:w,setDeleteTarget:S,load:P,openCreate:z,openEdit:B,save:H,toggleEnabled:W,confirmDelete:q}}(),z=function(){let e=(0,l.useTranslations)("adminPrompts"),{accessToken:t}=(0,A.useAuthSession)(),[a,i]=s.useState([]),[o,r]=s.useState(0),[c,d]=s.useState(1),[p,m]=s.useState(25),[u,h]=s.useState(""),[x,g]=s.useState(""),[f,_]=s.useState(!0),[y,v]=s.useState(!1),[b,k]=s.useState(E.EMPTY_SKILL_FORM),[j,w]=s.useState(!1),[C,S]=s.useState(null),[,T]=s.useTransition(),N=s.useRef(0);s.useEffect(()=>{let e=window.setTimeout(()=>{g(u.trim())},250);return()=>window.clearTimeout(e)},[u]);let P=s.useCallback(async()=>{let s=N.current+1;N.current=s,_(!0);try{let e=await (0,M.listAdminSkills)(t,{page:c,pageSize:p,query:x});if(s!==N.current)return;T(()=>{i(e.results),r(e.total)})}catch(t){n.toast.error(e("toast.skillsLoadFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}finally{s===N.current&&_(!1)}},[t,x,c,p,T,e]);s.useEffect(()=>{P()},[P]);let D=Math.max(1,Math.ceil(o/p)),I=s.useCallback(e=>{h(e),d(1)},[]),F=s.useCallback(e=>{m(e),d(1)},[]),R=s.useCallback(()=>{k(E.EMPTY_SKILL_FORM),w(!0)},[]),z=s.useCallback(e=>{k((0,E.skillFormFromDTO)(e)),w(!0)},[]),B=s.useCallback(async()=>{let s=(0,E.skillPayloadFromForm)(b);if(!(0,E.skillPayloadIsComplete)(s))return void n.toast.error(e("toast.skillInvalid"));if(!(0,E.skillFormIsWithinLimits)(b))return void n.toast.error(e("toast.skillTooLong"));v(!0);try{b.id?(await (0,M.updateAdminSkill)(t,b.id,s),await P(),n.toast.success(e("toast.skillUpdated"))):(await (0,M.createAdminSkill)(t,s),await P(),n.toast.success(e("toast.skillCreated"))),w(!1)}catch(t){n.toast.error(b.id?e("toast.skillUpdateFailed"):e("toast.skillCreateFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}finally{v(!1)}},[t,b,P,e]),H=s.useCallback(async(s,a)=>{i(e=>e.map(e=>e.id===s.id?{...e,enabled:a}:e));try{await (0,M.updateAdminSkill)(t,s.id,{enabled:a}),await P()}catch(t){i(e=>(0,L.replaceByID)(e,s.id,e=>e.id,s)),n.toast.error(e("toast.skillUpdateFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}},[t,P,e]),W=s.useCallback(async()=>{if(C){S(null);try{await (0,M.deleteAdminSkill)(t,C.id),i(e=>(0,L.removeByID)(e,C.id,e=>e.id)),r(e=>Math.max(0,e-1)),await P(),n.toast.success(e("toast.skillDeleted"))}catch(t){n.toast.error(e("toast.skillDeleteFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}}},[t,C,P,e]);return{items:a,total:o,page:c,pageSize:p,pageCount:D,query:u,loading:f,saving:y,form:b,dialogOpen:j,deleteTarget:C,setPage:d,setPageSize:F,setQuery:I,setForm:k,setDialogOpen:w,setDeleteTarget:S,load:P,openCreate:R,openEdit:z,save:B,toggleEnabled:H,confirmDelete:W}}(),K="skills"===i?z.loading:P.loading,V="skills"===i?z.query:P.query,U="skills"===i?z.page:P.page,G="skills"===i?z.pageCount:P.pageCount,Y="skills"===i?z.pageSize:P.pageSize,$="skills"===i?z.total:P.total,Q="skills"===i?e("skillsSearchPlaceholder"):e("searchPlaceholder"),J="skills"===i?e("createSkill"):e("create"),X=c!==p;s.useEffect(()=>{let t=!1;return(async()=>{f(!0);try{let s=await (0,B.resolveAccessToken)();if(!s)return void n.toast.error(e("toast.sessionExpired"),{description:e("toast.signInAgain")});let a=await (0,D.listAdminSettingsByNamespace)(s,"chat");if(t)return;let i=a.find(e=>"skills_prompt"===e.key)?.value??"";d(i),m(i)}catch(s){t||n.toast.error(e("toast.settingsLoadFailed"),{description:(0,O.resolveAdminErrorMessage)(s)})}finally{t||f(!1)}})(),()=>{t=!0}},[e]);let Z=s.useCallback(async()=>{if(X){S(!0);try{let t=await (0,B.resolveAccessToken)();if(!t)return void n.toast.error(e("toast.sessionExpired"),{description:e("toast.signInAgain")});let s=[{namespace:"chat",key:"skills_prompt",value:c}],a=await (0,D.patchAdminSettings)(t,{items:s}),i=a.chat?.find(e=>"skills_prompt"===e.key)?.value??c;d(i),m(i),n.toast.success(e("toast.settingsUpdated"))}catch(t){n.toast.error(e("toast.settingsSaveFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}finally{S(!1)}}},[X,c,e]),ee="skills"===i&&X?(0,t.jsxs)(v.Button,{type:"button",size:"sm",className:"h-7 gap-1 px-2.5 text-xs",disabled:u||_,onClick:()=>void Z(),children:[(0,t.jsx)(o.Save,{className:"size-3.5"}),a("actions.save")]}):null,et=(0,t.jsx)(C.Tabs,{value:i,onValueChange:e=>r(e),children:(0,t.jsxs)(C.TabsList,{children:[(0,t.jsx)(C.TabsTrigger,{value:"skills",children:e("types.skills")}),(0,t.jsx)(C.TabsTrigger,{value:"prompts",children:e("types.prompts")})]})});return(0,t.jsxs)(t.Fragment,{children:[(0,t.jsx)(H.SettingsSection,{title:e("title"),actions:et,children:(0,t.jsxs)("div",{className:"space-y-4",children:["skills"===i?(0,t.jsx)(W,{value:c,action:ee,dirty:X,disabled:u||_,onChange:d}):null,(0,t.jsxs)("div",{className:"space-y-2",children:[(0,t.jsx)("div",{className:"px-0.5",children:(0,t.jsx)("p",{className:"text-xs font-medium leading-snug text-foreground/80",children:"skills"===i?e("libraryTitle.skills"):e("libraryTitle.prompts")})}),(0,t.jsx)(T.TableToolbar,{query:V,queryPlaceholder:Q,onQueryChange:"skills"===i?z.setQuery:P.setQuery,loading:K,onRefresh:()=>void("skills"===i?z.load():P.load()),children:(0,t.jsxs)(v.Button,{type:"button",size:"sm",className:"h-7 gap-1 text-xs",onClick:"skills"===i?z.openCreate:P.openCreate,disabled:K,children:[(0,t.jsx)(g.Plus,{className:"size-3.5 stroke-1"}),J]})})]}),"skills"===i?(0,t.jsx)(q,{emptyLabel:e("skillsEmpty"),getSummary:e=>e.description||e.markdown,icon:h.Box,items:z.items,loading:z.loading,onEdit:z.openEdit,onDelete:z.setDeleteTarget,onEnabledChange:(e,t)=>void z.toggleEnabled(e,t)}):(0,t.jsx)(q,{emptyLabel:e("empty"),getSummary:e=>e.description||e.content,icon:x,items:P.items,loading:P.loading,onEdit:P.openEdit,onDelete:P.setDeleteTarget,onEnabledChange:(e,t)=>void P.toggleEnabled(e,t)}),(0,t.jsx)(T.TablePagination,{page:U,pageCount:G,pageSize:Y,total:$,onPageChange:"skills"===i?z.setPage:P.setPage,onPageSizeChange:"skills"===i?z.setPageSize:P.setPageSize,loading:K})]})}),(0,t.jsx)(b.Dialog,{open:P.dialogOpen,onOpenChange:e=>!P.saving&&P.setDialogOpen(e),children:(0,t.jsxs)(b.DialogContent,{className:"flex max-h-[min(86vh,760px)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[560px]",children:[(0,t.jsxs)(b.DialogHeader,{className:"shrink-0 px-5 pb-3 pt-5",children:[(0,t.jsx)(b.DialogTitle,{children:P.form.id?e("editTitle"):e("createTitle")}),(0,t.jsx)(b.DialogDescription,{children:e("dialogDescription")})]}),(0,t.jsxs)("div",{className:"min-h-0 flex-1 space-y-3 overflow-y-auto px-5 py-2",children:[(0,t.jsxs)("div",{className:"space-y-1",children:[(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:e("fields.name")}),(0,t.jsxs)(j.InputGroup,{children:[(0,t.jsx)(j.InputGroupAddon,{children:"/"}),(0,t.jsx)(j.InputGroupInput,{value:P.form.name,placeholder:"musk",maxLength:F.PROMPT_PRESET_LIMITS.name,onChange:e=>P.setForm(t=>({...t,name:e.target.value}))})]})]}),(0,t.jsxs)("div",{className:"space-y-1",children:[(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:e("fields.description")}),(0,t.jsx)(k.Input,{value:P.form.description,maxLength:F.PROMPT_PRESET_LIMITS.description,onChange:e=>P.setForm(t=>({...t,description:e.target.value}))})]}),(0,t.jsxs)("div",{className:"space-y-1",children:[(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:e("fields.content")}),(0,t.jsx)(N.Textarea,{value:P.form.content,className:"h-64 resize-none overflow-y-auto [field-sizing:fixed]",maxLength:F.PROMPT_PRESET_LIMITS.content,onChange:e=>P.setForm(t=>({...t,content:e.target.value}))})]}),(0,t.jsxs)("div",{className:"space-y-1",children:[(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:e("fields.enabled")}),(0,t.jsx)(w.Switch,{size:"sm",checked:P.form.enabled,disabled:P.saving,onCheckedChange:e=>P.setForm(t=>({...t,enabled:e}))})]})]}),(0,t.jsxs)(b.DialogFooter,{className:"shrink-0 px-5 py-3",children:[(0,t.jsx)(v.Button,{variant:"ghost",disabled:P.saving,onClick:()=>P.setDialogOpen(!1),children:e("cancel")}),(0,t.jsx)(v.Button,{disabled:P.saving,onClick:()=>void P.save(),children:P.saving?e("saving"):e("save")})]})]})}),(0,t.jsx)(b.Dialog,{open:z.dialogOpen,onOpenChange:e=>!z.saving&&z.setDialogOpen(e),children:(0,t.jsxs)(b.DialogContent,{className:"flex max-h-[min(86vh,760px)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[560px]",children:[(0,t.jsxs)(b.DialogHeader,{className:"shrink-0 px-5 pb-3 pt-5",children:[(0,t.jsx)(b.DialogTitle,{children:z.form.id?e("editSkillTitle"):e("createSkillTitle")}),(0,t.jsx)(b.DialogDescription,{children:e("skillDialogDescription")})]}),(0,t.jsxs)("div",{className:"min-h-0 flex-1 space-y-3 overflow-y-auto px-5 py-2",children:[(0,t.jsxs)("div",{className:"space-y-1",children:[(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:e("fields.name")}),(0,t.jsxs)(j.InputGroup,{children:[(0,t.jsx)(j.InputGroupAddon,{children:"/"}),(0,t.jsx)(j.InputGroupInput,{value:z.form.name,placeholder:"review",maxLength:E.SKILL_LIMITS.name,onChange:e=>z.setForm(t=>({...t,name:e.target.value}))})]})]}),(0,t.jsxs)("div",{className:"space-y-1",children:[(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:e("fields.description")}),(0,t.jsx)(k.Input,{value:z.form.description,maxLength:E.SKILL_LIMITS.description,onChange:e=>z.setForm(t=>({...t,description:e.target.value}))})]}),(0,t.jsxs)("div",{className:"space-y-1",children:[(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:e("fields.skillMarkdown")}),(0,t.jsx)(N.Textarea,{value:z.form.markdown,className:"h-64 resize-none overflow-y-auto [field-sizing:fixed]",maxLength:E.SKILL_LIMITS.markdown,onChange:e=>z.setForm(t=>({...t,markdown:e.target.value}))})]}),(0,t.jsxs)("div",{className:"space-y-1",children:[(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:e("fields.enabled")}),(0,t.jsx)(w.Switch,{size:"sm",checked:z.form.enabled,disabled:z.saving,onCheckedChange:e=>z.setForm(t=>({...t,enabled:e}))})]})]}),(0,t.jsxs)(b.DialogFooter,{className:"shrink-0 px-5 py-3",children:[(0,t.jsx)(v.Button,{variant:"ghost",disabled:z.saving,onClick:()=>z.setDialogOpen(!1),children:e("cancel")}),(0,t.jsx)(v.Button,{disabled:z.saving,onClick:()=>void z.save(),children:z.saving?e("saving"):e("save")})]})]})}),(0,t.jsx)(y.AlertDialog,{open:null!==P.deleteTarget,onOpenChange:e=>!e&&P.setDeleteTarget(null),children:(0,t.jsxs)(y.AlertDialogContent,{children:[(0,t.jsxs)(y.AlertDialogHeader,{children:[(0,t.jsx)(y.AlertDialogTitle,{children:e("deleteTitle")}),(0,t.jsx)(y.AlertDialogDescription,{children:e("deleteDescription")})]}),(0,t.jsxs)(y.AlertDialogFooter,{children:[(0,t.jsx)(y.AlertDialogCancel,{children:e("cancel")}),(0,t.jsx)(y.AlertDialogAction,{onClick:()=>void P.confirmDelete(),children:e("delete")})]})]})}),(0,t.jsx)(y.AlertDialog,{open:null!==z.deleteTarget,onOpenChange:e=>!e&&z.setDeleteTarget(null),children:(0,t.jsxs)(y.AlertDialogContent,{children:[(0,t.jsxs)(y.AlertDialogHeader,{children:[(0,t.jsx)(y.AlertDialogTitle,{children:e("deleteSkillTitle")}),(0,t.jsx)(y.AlertDialogDescription,{children:e("deleteSkillDescription")})]}),(0,t.jsxs)(y.AlertDialogFooter,{children:[(0,t.jsx)(y.AlertDialogCancel,{children:e("cancel")}),(0,t.jsx)(y.AlertDialogAction,{onClick:()=>void z.confirmDelete(),children:e("delete")})]})]})})]})}var V=e.i(413689),U=e.i(79923);let G="follow",Y={field:"chat.context_compact_enabled",equals:"true"},$={all:[Y,{field:"chat.compact_llm_enabled",equals:"true"}]},Q=`{
  "default": [
    "temperature",
    "top_p",
    "max_tokens",
    "max_output_tokens",
    "max_completion_tokens",
    "stop",
    "tools",
    "response_format.type"
  ],
  "openai_chat_completions": [
    "service_tier",
    "presence_penalty",
    "frequency_penalty",
    "reasoning_effort",
    "verbosity",
    "thinking.type",
    "stream_options.include_usage"
  ],
  "openrouter_chat_completions": [
    "presence_penalty",
    "frequency_penalty",
    "reasoning_effort",
    "reasoning.effort",
    "reasoning.summary",
    "verbosity",
    "thinking.type",
    "stream_options.include_usage"
  ],
  "openai_responses": [
    "service_tier",
    "store",
    "reasoning.effort",
    "reasoning.summary",
    "text.verbosity"
  ],
  "openai_image_generations": [
    "background",
    "moderation",
    "n",
    "output_compression",
    "output_format",
    "partial_images",
    "quality",
    "response_format",
    "size",
    "style",
    "user"
  ],
  "openai_image_edits": [
    "background",
    "input_fidelity",
    "n",
    "output_compression",
    "output_format",
    "partial_images",
    "quality",
    "response_format",
    "size",
    "user"
  ],
  "google_image_generation": [
    "generationConfig.responseModalities",
    "generationConfig.imageConfig.aspectRatio",
    "generationConfig.imageConfig.imageSize"
  ],
  "gemini_interactions": [
    "generation_config.temperature",
    "generation_config.top_p",
    "generation_config.max_output_tokens",
    "generation_config.thinking_level",
    "response_format.type",
    "response_format.aspect_ratio",
    "response_format.image_size",
    "response_format.mime_type",
    "responseFormat.type",
    "responseFormat.aspectRatio",
    "responseFormat.imageSize",
    "responseFormat.mimeType",
    "generationConfig.videoConfig.task",
    "generation_config.video_config.task"
  ],
  "anthropic_messages": [
    "speed",
    "top_k",
    "cache_control",
    "thinking.type",
    "thinking.budget_tokens"
  ],
  "xai_responses": [
    "store",
    "reasoning.effort"
  ],
  "xai_image": [
    "aspect_ratio",
    "n",
    "resolution",
    "response_format"
  ],
  "xai_image_edits": [
    "aspect_ratio",
    "n",
    "resolution",
    "response_format"
  ],
  "gemini_generate_content": [
    "generationConfig.temperature",
    "generationConfig.topP",
    "generationConfig.maxOutputTokens",
    "generationConfig.responseMimeType"
  ]
}`,J=`{
  "default": [
    "model",
    "messages",
    "input",
    "instructions",
    "prompt",
    "system",
    "systemInstruction",
    "headers",
    "api_key",
    "apiKey",
    "base_url",
    "baseURL",
    "stream",
    "previous_response_id"
  ]
}`;function X(e){return`${e.namespace}.${e.key}`}function Z(e){let t,s={};for(let t of e.chat??[])s[`chat.${t.key}`]=t.value??"";return((t={...s})["chat.conversation_task_model"]??"").trim()||(t["chat.conversation_task_model"]=G),(t["chat.model_option_policy_mode"]??"").trim()||(t["chat.model_option_policy_mode"]="allowlist"),(t["chat.model_option_allowed_paths"]??"").trim()||(t["chat.model_option_allowed_paths"]=Q),(t["chat.model_option_denied_paths"]??"").trim()||(t["chat.model_option_denied_paths"]=J),(t["chat.compact_task_model"]??"").trim()||(t["chat.compact_task_model"]=G),(t["chat.context_artifact_retention_days"]??"").trim()||(t["chat.context_artifact_retention_days"]="90"),t["chat.conversation_title_prompt"]=ee(t["chat.conversation_title_prompt"]??""),t["chat.conversation_labels_prompt"]=ee(t["chat.conversation_labels_prompt"]??""),t["chat.default_system_prompt"]=ee(t["chat.default_system_prompt"]??""),t["chat.compact_system_prompt"]=ee(t["chat.compact_system_prompt"]??""),t["chat.compact_light_prompt"]=ee(t["chat.compact_light_prompt"]??""),t}function ee(e){return e.trim()?e:""}var et=e.i(597916);function es({models:e,followLabel:t,followValue:s}){let a=new Set,i=[{label:t,value:s,iconUrl:null}];for(let t of e){if(!(0,et.isRoutableChatPlatformModel)(t))continue;let e=t.platformModelName.trim();!e||a.has(e)||(a.add(e),i.push({label:(0,et.resolveModelOptionLabel)(e),value:e,iconUrl:(0,et.resolveModelOptionIconUrl)({platformModelName:e,vendor:t.vendor??"",icon:t.icon??""})}))}return i}var ea=e.i(281639);function ei(e){return"optionPassthrough"===e.section}function eo(e,t){return ea.MODEL_OPTION_POLICY_PROTOCOLS.flatMap(s=>(0,ea.uniqueModelOptionPaths)(e[s]??[]).map(e=>({path:e,reason:t,scope:ea.MODEL_OPTION_POLICY_PROTOCOL_LABELS[s]})))}function el({mode:e,allowedPathsJSON:a,deniedPathsJSON:i,t:o}){let l=s.useMemo(()=>(0,ea.parseModelOptionRuleMap)(a),[a]),n=s.useMemo(()=>(0,ea.parseModelOptionRuleMap)(i),[i]),r=s.useMemo(()=>{let t=[...ea.HARD_DENIED_MODEL_OPTION_PATHS.map(e=>({path:e,reason:o("preview.systemDenied"),scope:"Default"})),..."denylist"===e?eo(n.value,o("preview.hitDenylist")).filter(e=>!ea.HARD_DENIED_MODEL_OPTION_PATHS.includes(e.path)):[]],s=new Set(t.map(e=>e.path));return{passedRows:"denylist"===e?ea.MODEL_OPTION_POLICY_PROTOCOLS.map(e=>({path:o("preview.otherFields"),reason:o("preview.notInDenylist"),scope:ea.MODEL_OPTION_POLICY_PROTOCOL_LABELS[e]})):eo(l.value,o("preview.hitAllowlist")).filter(e=>!s.has(e.path)),filteredRows:t}},[l.value,n.value,e,o]),c=l.error||n.error;return(0,t.jsxs)("div",{className:"mt-4 space-y-3",children:[(0,t.jsx)("p",{className:"text-xs font-medium text-foreground/80",children:o("preview.title")}),c?(0,t.jsx)("p",{className:"text-xs text-destructive",children:c}):(0,t.jsxs)("div",{className:"grid gap-3 md:grid-cols-2",children:[(0,t.jsx)(en,{title:o("preview.passed"),rows:r.passedRows,emptyText:o("preview.emptyPassed")}),(0,t.jsx)(en,{title:o("preview.filtered"),rows:r.filteredRows,emptyText:o("preview.emptyFiltered")})]})]})}function en({title:e,rows:s,emptyText:a}){return(0,t.jsxs)("div",{className:"space-y-2 rounded-md bg-muted/30 p-2.5",children:[(0,t.jsxs)("div",{className:"flex items-center justify-between",children:[(0,t.jsx)("p",{className:"text-xs font-medium text-foreground/80",children:e}),(0,t.jsx)("span",{className:"text-[11px] text-muted-foreground",children:s.length})]}),0===s.length?(0,t.jsx)("p",{className:"text-xs text-muted-foreground",children:a}):(0,t.jsx)("div",{className:"space-y-1.5",children:s.map(e=>(0,t.jsxs)("div",{className:"grid gap-0.5",children:[(0,t.jsxs)("div",{className:"flex min-w-0 items-center gap-2",children:[(0,t.jsx)("code",{className:"min-w-0 flex-1 truncate text-xs text-foreground",children:e.path}),(0,t.jsx)("span",{className:"shrink-0 text-[11px] text-muted-foreground",children:e.scope})]}),(0,t.jsx)("span",{className:"text-[11px] text-muted-foreground",children:e.reason})]},`${e.scope}-${e.path}-${e.reason}`))})]})}function er({t:e}){return(0,t.jsxs)(b.Dialog,{children:[(0,t.jsx)(b.DialogTrigger,{asChild:!0,children:(0,t.jsxs)(v.Button,{type:"button",variant:"ghost",size:"sm",className:"h-6 px-2 text-xs font-normal text-muted-foreground hover:text-foreground",children:[(0,t.jsx)(a.CircleHelp,{className:"size-3.5"}),e("guide.button")]})}),(0,t.jsxs)(b.DialogContent,{className:"flex max-h-[min(86vh,760px)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[720px]",children:[(0,t.jsxs)(b.DialogHeader,{className:"shrink-0 px-4 py-4",children:[(0,t.jsx)(b.DialogTitle,{children:e("guide.title")}),(0,t.jsx)(b.DialogDescription,{children:e("guide.description")})]}),(0,t.jsxs)("div",{className:"min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-2 text-sm text-muted-foreground",children:[(0,t.jsxs)("section",{className:"space-y-2",children:[(0,t.jsx)("h4",{className:"text-sm font-medium text-foreground",children:e("guide.pathTitle")}),(0,t.jsxs)("div",{className:"grid gap-3 md:grid-cols-2",children:[(0,t.jsxs)("div",{className:"space-y-1.5",children:[(0,t.jsx)("p",{className:"text-xs font-medium text-foreground",children:"options"}),(0,t.jsx)("pre",{className:"max-h-44 overflow-auto rounded-md bg-muted/50 p-3 text-xs text-foreground",children:`{
  "temperature": 0.7,
  "thinking": {
    "type": "enabled"
  },
  "generationConfig": {
    "safetySettings": {
      "threshold": "BLOCK_NONE"
    }
  }
}`})]}),(0,t.jsxs)("div",{className:"space-y-1.5",children:[(0,t.jsx)("p",{className:"text-xs font-medium text-foreground",children:e("guide.pathLabel")}),(0,t.jsx)("pre",{className:"max-h-44 overflow-auto rounded-md bg-muted/50 p-3 text-xs text-foreground",children:`temperature
thinking.type
generationConfig.safetySettings.threshold`})]})]}),(0,t.jsx)("p",{className:"text-xs",children:e("guide.pathDescription")})]}),(0,t.jsxs)("section",{className:"space-y-2",children:[(0,t.jsx)("h4",{className:"text-sm font-medium text-foreground",children:e("guide.strategyTitle")}),(0,t.jsxs)(C.Tabs,{defaultValue:"allowlist",className:"gap-3",children:[(0,t.jsxs)(C.TabsList,{children:[(0,t.jsx)(C.TabsTrigger,{value:"allowlist",children:e("policy.allowlist")}),(0,t.jsx)(C.TabsTrigger,{value:"denylist",children:e("policy.denylist")})]}),(0,t.jsxs)(C.TabsContent,{value:"allowlist",className:"space-y-2",children:[(0,t.jsx)("p",{className:"text-xs",children:e("guide.allowlistDescription")}),(0,t.jsx)("pre",{className:"max-h-48 overflow-auto rounded-md bg-muted/50 p-3 text-xs text-foreground",children:`{
  "default": [
    "temperature",
    "top_p",
    "stop"
  ],
  "openai_responses": [
    "service_tier",
    "reasoning.effort",
    "text.verbosity"
  ],
  "openai_image_generations": [
    "background",
    "moderation",
    "n",
    "output_compression",
    "output_format",
    "partial_images",
    "quality",
    "size",
    "response_format",
    "style",
    "user"
  ],
  "openai_image_edits": [
    "background",
    "input_fidelity",
    "n",
    "output_compression",
    "output_format",
    "partial_images",
    "quality",
    "response_format",
    "size",
    "user"
  ],
  "google_image_generation": [
    "generationConfig.responseModalities",
    "generationConfig.imageConfig.aspectRatio",
    "generationConfig.imageConfig.imageSize"
  ],
  "gemini_interactions": [
    "generation_config.temperature",
    "generation_config.top_p",
    "generation_config.max_output_tokens",
    "generation_config.thinking_level",
    "response_format.type",
    "response_format.aspect_ratio",
    "response_format.image_size",
    "response_format.mime_type",
    "responseFormat.type",
    "responseFormat.aspectRatio",
    "responseFormat.imageSize",
    "responseFormat.mimeType",
    "generationConfig.videoConfig.task",
    "generation_config.video_config.task"
  ],
  "xai_image": [
    "aspect_ratio",
    "n",
    "resolution",
    "response_format"
  ],
  "xai_image_edits": [
    "aspect_ratio",
    "n",
    "resolution",
    "response_format"
  ],
  "openai_chat_completions": [
    "service_tier",
    "thinking.type"
  ],
  "openrouter_chat_completions": [
    "reasoning_effort",
    "reasoning.effort",
    "thinking.type"
  ],
  "openrouter_responses": [
    "reasoning.effort",
    "reasoning.summary"
  ],
  "anthropic_messages": [
    "speed",
    "thinking.type",
    "thinking.budget_tokens"
  ]
}`}),(0,t.jsx)("p",{className:"text-xs",children:e("guide.openAIServiceTierNote")})]}),(0,t.jsxs)(C.TabsContent,{value:"denylist",className:"space-y-2",children:[(0,t.jsx)("p",{className:"text-xs",children:e("guide.denylistDescription")}),(0,t.jsx)("pre",{className:"max-h-48 overflow-auto rounded-md bg-muted/50 p-3 text-xs text-foreground",children:`{
  "default": [
    "headers",
    "api_key",
    "previous_response_id"
  ],
  "anthropic_messages": [
    "thinking.budget_tokens",
    "metadata.user_id"
  ]
}`})]})]})]}),(0,t.jsxs)("section",{className:"space-y-2",children:[(0,t.jsx)("h4",{className:"text-sm font-medium text-foreground",children:e("guide.protocolTitle")}),(0,t.jsx)("p",{className:"text-xs",children:e("guide.protocolDescription")}),(0,t.jsx)("div",{className:"flex flex-wrap gap-1.5",children:["default","openai_chat_completions","openrouter_chat_completions","openai_responses","openrouter_responses","openai_image_generations","openai_image_edits","google_image_generation","gemini_interactions","xai_image","xai_image_edits","anthropic_messages","xai_responses","gemini_generate_content"].map(e=>(0,t.jsx)("code",{className:"rounded-md bg-muted/60 px-2 py-1 text-xs text-foreground",children:e},e))})]}),(0,t.jsxs)("section",{className:"space-y-2",children:[(0,t.jsx)("h4",{className:"text-sm font-medium text-foreground",children:e("guide.systemDeniedTitle")}),(0,t.jsx)("p",{className:"text-xs",children:e("guide.systemDeniedDescription")}),(0,t.jsx)("div",{className:"flex flex-wrap gap-1.5",children:ea.HARD_DENIED_MODEL_OPTION_PATHS.map(e=>(0,t.jsx)("code",{className:"rounded-md bg-muted/60 px-2 py-1 text-xs text-foreground",children:e},e))})]})]}),(0,t.jsx)(b.DialogFooter,{className:"shrink-0 px-4 py-3",children:(0,t.jsx)(b.DialogClose,{asChild:!0,children:(0,t.jsx)(v.Button,{type:"button",children:e("guide.close")})})})]})]})}e.s(["AdminConversationSettingsPage",0,function(){let e=(0,l.useTranslations)("adminConversation"),a=(0,l.useTranslations)("common"),r=s.useMemo(()=>[{section:"conversation",namespace:"chat",key:"conversation_default_model",label:e("fields.defaultModel.label"),description:e("fields.defaultModel.description"),type:"select",options:[{label:e("defaultModel.systemRecommended"),value:""}]},{section:"conversation",namespace:"chat",key:"conversation_task_model",label:e("fields.taskModel.label"),description:e("fields.taskModel.description"),type:"select",options:[{label:e("taskModel.follow"),value:G}]},{section:"optionPassthrough",namespace:"chat",key:"model_option_policy_mode",label:e("fields.optionPolicyMode.label"),description:e("fields.optionPolicyMode.description"),type:"select",options:[{label:e("policy.allowlist"),value:"allowlist"},{label:e("policy.denylist"),value:"denylist"},{label:e("policy.disabled"),value:"disabled"}]},{section:"optionPassthrough",namespace:"chat",key:"model_option_allowed_paths",label:e("fields.allowedPaths.label"),description:e("fields.allowedPaths.description"),type:"json",placeholder:Q},{section:"optionPassthrough",namespace:"chat",key:"model_option_denied_paths",label:e("fields.deniedPaths.label"),description:e("fields.deniedPaths.description"),type:"json",placeholder:J},{section:"conversation",namespace:"chat",key:"conversation_title_prompt",label:e("fields.titlePrompt.label"),description:e("fields.titlePrompt.description"),type:"textarea",placeholder:e("fields.defaultPromptPlaceholder")},{section:"conversation",namespace:"chat",key:"conversation_labels_prompt",label:e("fields.labelsPrompt.label"),description:e("fields.labelsPrompt.description"),type:"textarea",placeholder:e("fields.defaultPromptPlaceholder")},{section:"conversation",namespace:"chat",key:"default_system_prompt",label:e("fields.defaultSystemPrompt.label"),description:e("fields.defaultSystemPrompt.description"),type:"textarea",placeholder:e("fields.defaultSystemPrompt.placeholder")},{section:"contextCompression",namespace:"chat",key:"context_compact_enabled",label:e("fields.contextCompactEnabled.label"),description:e("fields.contextCompactEnabled.description"),type:"bool"},{section:"contextCompression",namespace:"chat",key:"context_token_budget_enabled",label:e("fields.contextTokenBudget.label"),description:e("fields.contextTokenBudget.description"),type:"bool",visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"context_max_turns",label:e("fields.contextMaxTurns.label"),description:e("fields.contextMaxTurns.description"),type:"int",placeholder:e("fields.contextMaxTurns.placeholder"),visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"context_compact_trigger_tokens",label:e("fields.contextCompactTriggerTokens.label"),description:e("fields.contextCompactTriggerTokens.description"),type:"int",placeholder:e("fields.contextCompactTriggerTokens.placeholder"),visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"context_compact_preserve_recent_turns",label:e("fields.contextCompactPreserveTurns.label"),description:e("fields.contextCompactPreserveTurns.description"),type:"int",placeholder:e("fields.contextCompactPreserveTurns.placeholder"),visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"context_compact_highlights_per_role",label:e("fields.contextCompactHighlightsPerRole.label"),description:e("fields.contextCompactHighlightsPerRole.description"),type:"int",placeholder:e("fields.contextCompactHighlightsPerRole.placeholder"),visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"context_compact_snippet_chars",label:e("fields.contextCompactSnippetChars.label"),description:e("fields.contextCompactSnippetChars.description"),type:"int",placeholder:e("fields.contextCompactSnippetChars.placeholder"),visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"context_artifact_retention_days",label:e("fields.contextArtifactRetentionDays.label"),description:e("fields.contextArtifactRetentionDays.description"),type:"int",placeholder:e("fields.contextArtifactRetentionDays.placeholder"),visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"compact_async_enabled",label:e("fields.compactAsync.label"),description:e("fields.compactAsync.description"),type:"bool",visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"compact_llm_enabled",label:e("fields.compactLLM.label"),description:e("fields.compactLLM.description"),type:"bool",visibleWhen:Y},{section:"contextCompression",namespace:"chat",key:"compact_task_model",label:e("fields.compactTaskModel.label"),description:e("fields.compactTaskModel.description"),type:"select",options:[{label:e("taskModel.follow"),value:G}],visibleWhen:$,subgroupKey:"compact_llm"},{section:"contextCompression",namespace:"chat",key:"compact_max_failures",label:e("fields.compactMaxFailures.label"),description:e("fields.compactMaxFailures.description"),type:"int",placeholder:e("fields.compactMaxFailures.placeholder"),visibleWhen:$,subgroupKey:"compact_llm"},{section:"contextCompression",namespace:"chat",key:"compact_system_prompt",label:e("fields.compactSystemPrompt.label"),description:e("fields.compactSystemPrompt.description"),type:"textarea",placeholder:e("fields.defaultPromptPlaceholder"),visibleWhen:$,subgroupKey:"compact_llm"},{section:"contextCompression",namespace:"chat",key:"compact_light_prompt",label:e("fields.compactLightPrompt.label"),description:e("fields.compactLightPrompt.description"),type:"textarea",placeholder:e("fields.defaultPromptPlaceholder"),visibleWhen:$,subgroupKey:"compact_llm"}],[e]),[c,d]=s.useState(!0),[p,h]=s.useState(!1),[x,g]=s.useState({}),[f,_]=s.useState({}),[y,b]=s.useState(()=>es({models:[],followLabel:e("taskModel.follow"),followValue:G})),[k,j]=s.useState(()=>es({models:[],followLabel:e("defaultModel.systemRecommended"),followValue:""})),[w,C]=s.useState(!1),S=s.useCallback(async()=>{C(!0);try{let t=await (0,B.resolveAccessToken)();if(!t)return;let{blob:s,fileName:a}=await (0,D.exportAllConversations)(t),i=await (0,V.readExportManifest)(s);(0,V.downloadBlob)(s,a),i&&(!i.complete||(i.failed??0)>0)?n.toast.warning(e("dataExport.partial",{exported:i.exported??0,failed:i.failed??0})):i&&n.toast.success(e("dataExport.success",{count:i.exported??0}))}catch{n.toast.error(e("dataExport.failed"))}finally{C(!1)}},[e]),T=s.useCallback(async()=>{d(!0);try{let t=await (0,B.resolveAccessToken)();if(!t)return void n.toast.error(e("toast.sessionExpired"),{description:e("toast.signInAgain")});let[s,a]=await Promise.all([(0,D.listAdminSettings)(t),(0,U.getAdminReferenceData)(t).catch(()=>null)]),i=es({models:a?.models??[],followLabel:e("taskModel.follow"),followValue:G}),o=es({models:a?.models??[],followLabel:e("defaultModel.systemRecommended"),followValue:""}),l=Z(s);b(i),j(o),g(l),_(l)}catch(t){n.toast.error(e("toast.loadFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}finally{d(!1)}},[e]);s.useEffect(()=>{T()},[T]);let N=s.useMemo(()=>{let e=new Set;for(let t of r){let s=X(t);(x[s]??"")!==(f[s]??"")&&e.add(s)}return e},[r,f,x]),P=s.useCallback(async t=>{let s=t.filter(e=>N.has(X(e))).map(e=>({namespace:e.namespace,key:e.key,value:x[X(e)]??""}));if(0!==s.length){h(!0);try{let t=await (0,B.resolveAccessToken)();if(!t)return void n.toast.error(e("toast.sessionExpired"),{description:e("toast.signInAgain")});let a=await (0,D.patchAdminSettings)(t,{items:s}),i=Z(a);g(i),_(i),n.toast.success(e("toast.updated"))}catch(t){n.toast.error(e("toast.saveFailed"),{description:(0,O.resolveAdminErrorMessage)(t)})}finally{h(!1)}}},[N,x,e]),M=s.useMemo(()=>r.filter(e=>(function e(t,s){return!t||("all"in t?t.all.every(t=>e(t,s)):(s[t.field]??"")===t.equals)})(e.visibleWhen,x)),[r,x]),A=s.useMemo(()=>M.filter(e=>"conversation"===e.section),[M]),L=s.useMemo(()=>M.filter(e=>"contextCompression"===e.section),[M]),E=s.useMemo(()=>M.filter(ei),[M]),I=x["chat.model_option_policy_mode"]||"allowlist",F=s.useMemo(()=>E.find(e=>"model_option_policy_mode"===e.key)??null,[E]),R=s.useMemo(()=>{let e="denylist"===I?"model_option_denied_paths":"allowlist"===I?"model_option_allowed_paths":"";return e?E.find(t=>t.key===e)??null:null},[E,I]),z=s.useCallback(e=>e.some(e=>N.has(X(e))),[N]),W=s.useCallback(e=>z(e)?(0,t.jsxs)(v.Button,{type:"button",size:"sm",disabled:c||p,onClick:()=>void P(e),children:[(0,t.jsx)(o.Save,{className:"size-3.5"}),a("actions.save")]}):null,[a,P,z,c,p]),q=W(E),ee=W(L),et=W(A);function ea(s,a,i){let o=X(s),l="model_option_allowed_paths"===s.key||"model_option_denied_paths"===s.key?(0,t.jsx)(er,{t:e}):void 0,n="model_option_allowed_paths"===s.key||"model_option_denied_paths"===s.key?(0,t.jsx)(el,{mode:I,allowedPathsJSON:x["chat.model_option_allowed_paths"]??"",deniedPathsJSON:x["chat.model_option_denied_paths"]??"",t:e}):void 0,r="chat.conversation_default_model"===o?(0,t.jsx)(m,{id:o,label:s.label,description:s.description,value:x[o]??"",fallbackValue:"",dirty:(x[o]??"")!==(f[o]??""),disabled:c||p,modelOptions:k,onChange:e=>g(t=>({...t,[o]:e}))}):"chat.conversation_task_model"===o||"chat.compact_task_model"===o?(0,t.jsx)(m,{id:o,label:s.label,description:s.description,value:x[o]??"",fallbackValue:G,dirty:(x[o]??"")!==(f[o]??""),disabled:c||p,modelOptions:y,onChange:e=>g(t=>({...t,[o]:e}))}):(0,t.jsx)(u.SettingsFieldEditor,{field:{id:X(s),label:s.label,description:s.description,type:s.type,placeholder:s.placeholder,options:s.options},value:x[o]??"",dirty:(x[o]??"")!==(f[o]??""),disabled:c||p,labelAction:l,afterControl:n,animateLayout:i?.animateLayout??!0,onChange:e=>g(t=>({...t,[o]:e}))});return(0,t.jsx)(H.SettingsFieldItem,{index:a,children:i?.inset?(0,t.jsx)(H.SettingsFieldInset,{children:r}):r},o)}return(0,t.jsxs)(H.SettingsPage,{children:[(0,t.jsx)(H.SettingsSection,{title:e("sections.conversation"),actions:et,children:(0,t.jsx)(H.SettingsFieldList,{children:A.map((e,t)=>ea(e,t))})}),(0,t.jsx)(H.SettingsSectionSeparator,{}),(0,t.jsx)(H.SettingsSection,{title:e("sections.contextCompression"),actions:ee,children:(0,t.jsx)(H.SettingsFieldList,{children:L.map((e,t)=>ea(e,t,{inset:!!e.subgroupKey}))})}),(0,t.jsx)(H.SettingsSectionSeparator,{}),(0,t.jsx)(K,{}),(0,t.jsx)(H.SettingsSectionSeparator,{}),(0,t.jsx)(H.SettingsSection,{title:e("sections.optionPassthrough"),actions:q,children:(0,t.jsxs)(H.SettingsFieldList,{children:[F?ea(F,0):null,R?ea(R,1,{animateLayout:!1}):null]})}),(0,t.jsx)(H.SettingsSectionSeparator,{}),(0,t.jsx)(H.SettingsSection,{title:e("sections.dataExport"),children:(0,t.jsx)(H.SettingsFieldList,{children:(0,t.jsx)(H.SettingsFieldItem,{children:(0,t.jsx)(H.SettingsFieldRow,{title:e("dataExport.title"),description:e("dataExport.description"),children:(0,t.jsxs)(v.Button,{variant:"default",size:"sm",disabled:w,onClick:S,children:[(0,t.jsx)(i.Download,{className:"size-3.5"}),e("dataExport.exportButton")]})})})})})]})}],784669)}]);