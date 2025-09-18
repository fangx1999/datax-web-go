// common.js - 通用前端工具函数
// 用于减少各页面间的重复代码

/**
 * 创建表格搜索筛选功能
 * @param {string} tableId - 表格ID
 * @param {string} searchId - 搜索输入框ID
 * @param {string} filterId - 筛选下拉框ID（可选）
 * @param {Object} options - 配置选项
 */
function createTableFilter(tableId, searchId, filterId, options = {}) {
  const table = document.getElementById(tableId);
  const search = document.getElementById(searchId);
  const filter = filterId ? document.getElementById(filterId) : null;
  
  if (!table || !search) {
    console.warn('Table filter: required elements not found');
    return;
  }

  const config = {
    searchFields: ['name', 'id'], // 默认搜索字段
    filterField: 'type', // 默认筛选字段
    emptyText: '暂无数据',
    debounceMs: 150,
    ...options
  };

  function updateEmptyState() {
    const tbody = table.tBodies[0];
    if (!tbody) return;

    const visibleRows = Array.from(tbody.querySelectorAll('tr[data-id]')).filter(tr => 
      tr.style.display !== 'none'
    );
    
    let emptyRow = tbody.querySelector('tr[data-empty]');
    if (visibleRows.length === 0) {
      if (!emptyRow) {
        emptyRow = document.createElement('tr');
        emptyRow.setAttribute('data-empty', '');
        const td = document.createElement('td');
        td.className = 'empty';
        td.textContent = config.emptyText;
        td.colSpan = (table.tHead && table.tHead.rows[0].cells.length) || 1;
        emptyRow.appendChild(td);
        tbody.appendChild(emptyRow);
      }
      emptyRow.style.display = '';
    } else if (emptyRow) {
      emptyRow.style.display = 'none';
    }
  }

  function applyFilter() {
    const keyword = (search.value || '').toLowerCase().trim();
    const filterValue = filter ? (filter.value || '').toLowerCase() : '';
    
    const rows = table.tBodies[0].rows;
    for (let i = 0; i < rows.length; i++) {
      const row = rows[i];
      if (!row.dataset || row.hasAttribute('data-empty')) continue;

      let matchesSearch = true;
      let matchesFilter = true;

      // 搜索匹配
      if (keyword) {
        matchesSearch = config.searchFields.some(field => {
          const value = (row.dataset[field] || '').toLowerCase();
          return value.includes(keyword);
        });
      }

      // 筛选匹配
      if (filter && filterValue) {
        const fieldValue = (row.dataset[config.filterField] || '').toLowerCase();
        matchesFilter = fieldValue === filterValue;
      }

      row.style.display = (matchesSearch && matchesFilter) ? '' : 'none';
    }
    
    updateEmptyState();
  }

  // 防抖函数
  function debounce(fn, ms) {
    let timeout;
    return function() {
      clearTimeout(timeout);
      const args = arguments;
      timeout = setTimeout(() => fn.apply(this, args), ms);
    };
  }

  // 绑定事件
  search.addEventListener('input', debounce(applyFilter, config.debounceMs));
  if (filter) {
    filter.addEventListener('change', applyFilter);
  }

  // 初始化
  applyFilter();
}

/**
 * 通用模态框控制
 */
const ModalManager = {
  show(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
      modal.style.display = 'flex';
      document.documentElement.style.overflow = 'hidden';
      // 聚焦到第一个输入框
      const firstInput = modal.querySelector('input, select, textarea');
      if (firstInput) firstInput.focus();
    }
  },

  hide(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
      modal.style.display = 'none';
      document.documentElement.style.overflow = '';
    }
  },

  init(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;

    // 点击背景关闭
    modal.addEventListener('click', (e) => {
      if (e.target === modal) {
        this.hide(modalId);
      }
    });

    // ESC键关闭
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && modal.style.display === 'flex') {
        this.hide(modalId);
      }
    });

    // 关闭按钮
    const closeBtn = modal.querySelector('.close, .close-x');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => this.hide(modalId));
    }
  },

  // 添加通用的模态框创建函数
  createModal(id, title, content, actions = []) {
    const modal = document.createElement('div');
    modal.id = id;
    modal.className = 'modal';
    modal.style.display = 'none';
    
    const actionsHtml = actions.map(action => 
      `<button class="btn ${action.class || ''}" onclick="${action.onclick}">${action.text}</button>`
    ).join('');
    
    modal.innerHTML = `
      <div class="modal-content">
        <div class="modal-header">
          <h3>${title}</h3>
          <button class="close" onclick="ModalManager.hide('${id}')">&times;</button>
        </div>
        <div class="modal-body">
          ${content}
        </div>
        <div class="modal-footer">
          ${actionsHtml}
        </div>
      </div>
    `;
    
    document.body.appendChild(modal);
    this.init(id);
    return modal;
  }
};


/**
 * 通用确认对话框
 */
function confirmAction(message, callback) {
  if (confirm(message)) {
    callback();
  }
}

/**
 * 通用AJAX请求函数
 */
async function apiRequest(url, options = {}) {
  const defaultOptions = {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'same-origin',
  };

  const mergedOptions = { ...defaultOptions, ...options };
  
  try {
    const response = await fetch(url, mergedOptions);
    const data = await response.json();
    
    if (!response.ok) {
      throw new Error(data.error || data.message || `HTTP ${response.status}`);
    }
    
    return { success: true, data };
  } catch (error) {
    console.error('API request failed:', error);
    return { success: false, error: error.message };
  }
}

/**
 * 通用状态切换功能（用于启用/禁用等操作）
 */
function createToggleHandler(buttonSelector, apiUrl, options = {}) {
  const config = {
    successMessage: '操作成功',
    errorMessage: '操作失败',
    ...options
  };

  document.addEventListener('click', async (e) => {
    const button = e.target.closest(buttonSelector);
    if (!button) return;

    e.preventDefault();
    if (button.disabled) return;

    const originalText = button.textContent;
    button.disabled = true;
    button.textContent = '处理中...';

    try {
      // 从按钮的data属性中获取ID，并替换API URL中的占位符
      const id = button.getAttribute('data-toggle-user');
      if (!id) {
        console.error('No user ID found in data-toggle-user attribute');
        button.textContent = originalText;
        button.disabled = false;
        return;
      }
      
      const actualUrl = apiUrl.replace('{id}', id);
      const result = await apiRequest(actualUrl, { method: 'POST' });
      
      if (result.success) {
        // 更新按钮状态
        const currentState = button.getAttribute('data-current');
        const newState = currentState === '1' ? '0' : '1';
        button.setAttribute('data-current', newState);
        
        // 更新按钮文本
        button.textContent = newState === '1' ? config.disableText : config.enableText;
        
        // 更新相关UI元素
        if (config.updateUI) {
          config.updateUI(button, newState);
        }
        
        // 重新应用筛选（如果有）
        if (config.refreshFilter) {
          config.refreshFilter();
        }
      } else {
        alert(config.errorMessage + ': ' + result.error);
        button.textContent = originalText;
      }
    } catch (error) {
      alert(config.errorMessage + ': ' + error.message);
      button.textContent = originalText;
    } finally {
      button.disabled = false;
    }
  });
}

/**
 * 通用表单验证
 */
function validateForm(form, rules = {}) {
  const errors = [];
  
  for (const [fieldName, rule] of Object.entries(rules)) {
    const field = form.querySelector(`[name="${fieldName}"]`);
    if (!field || field.disabled) continue;

    const value = field.value.trim();
    
    if (rule.required && !value) {
      errors.push(`${rule.label || fieldName} 不能为空`);
    }
    
    if (rule.minLength && value.length < rule.minLength) {
      errors.push(`${rule.label || fieldName} 至少需要 ${rule.minLength} 个字符`);
    }
    
    if (rule.maxLength && value.length > rule.maxLength) {
      errors.push(`${rule.label || fieldName} 不能超过 ${rule.maxLength} 个字符`);
    }
    
    if (rule.pattern && !rule.pattern.test(value)) {
      errors.push(`${rule.label || fieldName} 格式不正确`);
    }
  }
  
  return errors;
}

/**
 * 统一验证规则配置
 */
const ValidationRules = {
  // 通用字段规则
  name: { required: true, minLength: 1, maxLength: 100, label: '名称' },
  type: { required: true, label: '类型' },
  username: { required: true, minLength: 3, maxLength: 64, pattern: /^[a-zA-Z0-9_-]+$/, label: '用户名' },
  password: { required: true, minLength: 6, maxLength: 128, label: '密码' },
  role: { required: true, label: '角色' },
  cron: { required: true, minLength: 1, maxLength: 100, label: 'Cron表达式' },
  
  // 数据源字段规则
  db_url: { required: true, minLength: 1, maxLength: 255, pattern: /^[a-zA-Z0-9.-]+:\d+$|^[a-zA-Z]+:\/\/[a-zA-Z0-9.-]+:\d+$/, label: '数据库地址' },
  db_user: { required: true, minLength: 1, maxLength: 50, label: '数据库用户名' },
  db_password: { required: true, minLength: 1, maxLength: 100, label: '数据库密码' },
  db_database: { required: true, minLength: 1, maxLength: 100, label: '数据库名' },
  defaultfs: { required: true, minLength: 1, maxLength: 255, label: 'DefaultFS' },
  
  // 任务字段规则
  datax_json: { required: true, minLength: 1, label: 'DataX配置' },
  source_id: { required: true, label: '源数据源' },
  target_id: { required: true, label: '目标数据源' },
  flow_id: { required: true, label: '任务流' }
};

/**
 * 数据源表单校验
 */
function validateDataSourceForm(form) {
  const type = form.querySelector('[name="type"]')?.value;
  const rules = { name: ValidationRules.name, type: ValidationRules.type };
  
  if (type === 'mysql') {
    Object.assign(rules, {
      db_url: ValidationRules.db_url,
      db_user: ValidationRules.db_user,
      db_password: ValidationRules.db_password,
      db_database: ValidationRules.db_database
    });
  } else if (['ofs', 'hdfs', 'cosn'].includes(type)) {
    rules.defaultfs = ValidationRules.defaultfs;
  }
  
  return validateForm(form, rules);
}

/**
 * 通用表单提交处理器
 */
function createFormSubmitHandler(validationFunction, options = {}) {
  return function(e) {
    e.preventDefault();
    
    const form = e.target;
    const errors = validationFunction(form);
    
    if (errors.length > 0) {
      if (options.showErrors) {
        options.showErrors(errors, form);
      } else {
        alert(errors.join('\n'));
      }
      return false;
    }
    
    if (options.beforeSubmit) {
      if (!options.beforeSubmit(form)) {
        return false;
      }
    }
    
    form.submit();
  };
}

/**
 * 通用表单校验 - 根据类型自动选择规则
 */
function validateFormByType(form, type) {
  const rules = {};
  
  switch(type) {
    case 'task':
      Object.assign(rules, {
        name: ValidationRules.name,
        datax_json: ValidationRules.datax_json,
        source_id: ValidationRules.source_id,
        target_id: ValidationRules.target_id,
        flow_id: ValidationRules.flow_id
      });
      break;
    case 'user':
      Object.assign(rules, {
        username: ValidationRules.username,
        password: ValidationRules.password,
        role: ValidationRules.role
      });
      break;
    case 'taskFlow':
      Object.assign(rules, {
        name: ValidationRules.name,
        cron: ValidationRules.cron
      });
      break;
  }
  
  return validateForm(form, rules);
}

/**
 * 显示错误消息 - 简化版本
 */
function showErrors(errors, container) {
  const errorContainer = container || document.querySelector('.error-container');
  if (errorContainer) {
    errorContainer.innerHTML = errors.map(error => `<div class="alert error">${error}</div>`).join('');
  }
}

/**
 * 统一数据源字段命名处理
 * 解决后端使用指针类型和前端模板命名不一致的问题
 */
function normalizeDataSourceFields(ds) {
  return {
    id: ds.id || ds.ID,
    name: ds.name || ds.Name,
    type: ds.type || ds.Type,
    db_url: ds.db_url || ds.DBURL,
    db_user: ds.db_user || ds.DBUser,
    db_database: ds.db_database || ds.DBDatabase,
    db_password: ds.db_password || ds.DBPassword,
    defaultfs: ds.defaultfs || ds.DefaultFS,
    hadoopconfig: ds.hadoopconfig || ds.HadoopConfig
  };
}

/**
 * 统一的表单错误显示函数
 */
function showFormErrors(errors, form) {
  // 清除之前的错误提示
  form.querySelectorAll('.field-error').forEach(el => el.remove());
  
  if (errors.length > 0) {
    errors.forEach(error => {
      const fieldName = error.split(' ')[0];
      const field = form.querySelector(`[name="${fieldName}"]`);
      if (field) {
        const errorDiv = document.createElement('div');
        errorDiv.className = 'field-error';
        errorDiv.textContent = error;
        errorDiv.style.color = 'var(--error)';
        errorDiv.style.fontSize = '12px';
        errorDiv.style.marginTop = '4px';
        field.parentNode.appendChild(errorDiv);
      }
    });
    return false;
  }
  return true;
}


/**
 * 简单确认对话框
 */
function showConfirmDialog(message, onConfirm, onCancel) {
  // 创建遮罩层
  const overlay = document.createElement('div');
  overlay.style.cssText = `
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
    z-index: 10000;
    display: flex;
    align-items: center;
    justify-content: center;
  `;
  
  // 创建对话框
  const dialog = document.createElement('div');
  dialog.style.cssText = `
    background: white;
    border-radius: 8px;
    padding: 20px;
    max-width: 350px;
    width: 90%;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  `;
  
  dialog.innerHTML = `
    <div style="margin-bottom: 16px;">
      <p style="margin: 0; font-size: 14px; color: #333;">${message}</p>
    </div>
    <div style="display: flex; gap: 10px; justify-content: flex-end;">
      <button id="confirmCancel" style="
        padding: 6px 12px;
        border: 1px solid #ccc;
        background: white;
        color: #333;
        border-radius: 4px;
        cursor: pointer;
        font-size: 13px;
      ">取消</button>
      <button id="confirmOk" style="
        padding: 6px 12px;
        border: 1px solid #d32f2f;
        background: #d32f2f;
        color: white;
        border-radius: 4px;
        cursor: pointer;
        font-size: 13px;
      ">删除</button>
    </div>
  `;
  
  overlay.appendChild(dialog);
  document.body.appendChild(overlay);
  
  const cancelBtn = dialog.querySelector('#confirmCancel');
  const okBtn = dialog.querySelector('#confirmOk');
  
  // 事件处理
  const cleanup = () => {
    if (overlay.parentNode) {
      overlay.parentNode.removeChild(overlay);
    }
  };
  
  cancelBtn.addEventListener('click', () => {
    cleanup();
    if (onCancel) onCancel();
  });
  
  okBtn.addEventListener('click', () => {
    cleanup();
    if (onConfirm) onConfirm();
  });
  
  // ESC 键取消
  const handleKeydown = (e) => {
    if (e.key === 'Escape') {
      cleanup();
      if (onCancel) onCancel();
      document.removeEventListener('keydown', handleKeydown);
    }
  };
  document.addEventListener('keydown', handleKeydown);
  
  // 点击遮罩取消
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay) {
      cleanup();
      if (onCancel) onCancel();
    }
  });
}

/**
 * 通用确认删除函数 - 使用自定义对话框
 */
function confirmDelete(itemName = '该项目') {
  return new Promise((resolve) => {
    showConfirmDialog(`确定删除${itemName}?`, () => {
      resolve(true);
    }, () => {
      resolve(false);
    });
  });
}

// 全局函数，供模板调用
window.createTableFilter = createTableFilter;
window.ModalManager = ModalManager;
window.confirmAction = confirmAction;
window.apiRequest = apiRequest;
window.createToggleHandler = createToggleHandler;
window.validateForm = validateForm;
window.validateDataSourceForm = validateDataSourceForm;
window.validateFormByType = validateFormByType;
window.createFormSubmitHandler = createFormSubmitHandler;
window.showErrors = showErrors;
window.confirmDelete = confirmDelete;
window.normalizeDataSourceFields = normalizeDataSourceFields;
window.showFormErrors = showFormErrors;

// ========== 公共初始化函数 ==========


/**
 * 初始化删除按钮功能
 * @param {string} tableId - 表格ID
 * @param {string} deleteUrl - 删除API URL模板，使用 {id} 占位符
 * @param {string} entityName - 实体名称（用于确认对话框）
 */
function initDeleteButtons(tableId, deleteUrl, entityName) {
  const table = document.getElementById(tableId);
  if (!table) {
    console.warn('Delete buttons: table not found', { tableId });
    return;
  }

  table.addEventListener('click', function (e) {
    const deleteBtn = e.target.closest('.js-delete');
    if (deleteBtn) {
      e.preventDefault();
      const id = deleteBtn.dataset.id;
      const name = deleteBtn.dataset.name;
      
      if (!id) return;
      
      // 使用自定义确认对话框
      confirmDelete(entityName + ' "' + name + '"')
        .then(function(confirmed) {
          if (confirmed) {
            // 使用fetch API发送DELETE请求
            fetch(deleteUrl.replace('{id}', id), {
              method: 'DELETE',
              headers: {
                'Content-Type': 'application/json',
              },
              redirect: 'follow'  // 跟随重定向
            })
            .then(response => {
              if (response.ok) {
                return response.json();
              } else {
                return response.json().then(data => {
                  throw new Error(data.error || '删除失败');
                });
              }
            })
            .then(data => {
              if (data.redirect) {
                // 删除成功，重定向到指定页面
                window.location.href = data.redirect;
              } else {
                // 删除成功，刷新当前页面
                window.location.reload();
              }
            })
            .catch(error => {
              console.error('删除请求失败:', error);
              alert('删除失败：' + error.message);
            });
          }
        });
    }
  });
}

// 导出到全局作用域
window.initDeleteButtons = initDeleteButtons;
