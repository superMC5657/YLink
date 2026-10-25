import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, useDialog } from 'naive-ui'

/**
 * 非分页 CRUD 脚手架深模块(公告/知识库/套餐/优惠券共用,参照 useAdminList 的收敛方式):
 * loading/list/modal/editingId/saving/form + openCreate/openEdit/save/remove 一处归属。
 *
 * 收敛前四份视图各抄一份骨架且表单缺省字面量写两遍(reactive 初始化 + openCreate);
 * 本模块以 emptyForm 为缺省唯一来源,两处共用。域语义(字段校验文案、删除确认文案、
 * 弹窗打开时的附加装配如知识库分类联动)经回调留在各视图。
 * 提示/确认标题传 i18n key,由本模块统一 t(避免构造期快照 locale);
 * 依条目拼装的 content 回调在触发时求值,视图内 t() 无快照问题。
 */
export function useAdminCrud<TItem extends { id: number }, TForm extends object>(options: {
  /** 列表取数;需要旁路拉取伴生数据(如套餐页的分组)时在闭包内赋值后只返回本域列表 */
  fetch: () => Promise<{ list: TItem[] }>
  /** 新建态表单缺省(唯一来源:reactive 初始化与 openCreate 共用) */
  emptyForm: () => TForm
  /** 编辑态:行条目 → 表单 */
  toForm: (item: TItem) => TForm
  create: (body: TForm) => Promise<unknown>
  update: (id: number, body: TForm) => Promise<unknown>
  /** 保存前归一化(如知识库分类下拉 → category/category_id),先于 validate */
  beforeSave?: (form: TForm) => void
  /** 保存前校验:返回 false 中止(警告提示由调用方负责) */
  validate?: (form: TForm) => boolean
  /** 弹窗打开时的附加装配(表单填充后、显示前);item=null 为新建态 */
  onOpen?: (form: TForm, item: TItem | null) => void
  /** 删除传输(按条目 id) */
  remove: (id: number) => Promise<unknown>
  /** 删除确认弹窗:titleKey/positiveKey 为 i18n key,content 依条目拼装 */
  removeTexts: { titleKey: string; content: (item: TItem) => string; positiveKey: string }
  /** 成功提示 i18n key(created/updated 为保存,deleted 为删除) */
  messages: { created: string; updated: string; deleted: string }
}) {
  const { t } = useI18n()
  const message = useMessage()
  const dialog = useDialog()

  const loading = ref(false)
  const list = ref<TItem[]>([])
  const modal = ref(false)
  const editingId = ref<number | null>(null)
  const saving = ref(false)
  // reactive 后仍是同一对象;回调签名按 TForm 收,断言只做类型层面还原
  const form = reactive(options.emptyForm()) as TForm

  async function load() {
    loading.value = true
    try {
      const res = await options.fetch()
      list.value = res.list
    } finally {
      loading.value = false
    }
  }

  function openCreate() {
    editingId.value = null
    Object.assign(form, options.emptyForm())
    options.onOpen?.(form, null)
    modal.value = true
  }

  function openEdit(item: TItem) {
    editingId.value = item.id
    Object.assign(form, options.toForm(item))
    options.onOpen?.(form, item)
    modal.value = true
  }

  async function save() {
    options.beforeSave?.(form)
    if (options.validate && !options.validate(form)) return
    saving.value = true
    try {
      if (editingId.value === null) {
        await options.create({ ...form })
        message.success(t(options.messages.created))
      } else {
        await options.update(editingId.value, { ...form })
        message.success(t(options.messages.updated))
      }
      modal.value = false
      void load()
    } finally {
      saving.value = false
    }
  }

  function remove(item: TItem) {
    dialog.warning({
      title: t(options.removeTexts.titleKey),
      content: options.removeTexts.content(item),
      positiveText: t(options.removeTexts.positiveKey),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        await options.remove(item.id)
        message.success(t(options.messages.deleted))
        void load()
      },
    })
  }

  return { loading, list, modal, editingId, saving, form, load, openCreate, openEdit, save, remove }
}
