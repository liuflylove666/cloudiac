import { BellOutlined, CloudServerOutlined, CodeOutlined, DashboardOutlined, DollarOutlined, LayoutOutlined, InteractionOutlined, SettingOutlined, ProjectOutlined, FormOutlined, PlusSquareOutlined, SearchOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import getPermission from "utils/permission";

const getMenus = (userInfo, { projectList }) => {
  const { ORG_SET, PROJECT_SET } = getPermission(userInfo);
  return [
    {
      subName: '项目信息',
      subKey: 'project',
      isHide: !PROJECT_SET && projectList.length === 0,
      emptyMenuList: [
        {
          name: '创建项目',
          key: 'm-project-create',
          icon: <PlusSquareOutlined />
        }
      ],
      menuList: [
        {
          name: '环境',
          key: 'm-project-env',
          icon: <CodeOutlined />
        },
        {
          name: '云模板',
          key: 'm-project-ct',
          icon: <LayoutOutlined />
        },
        {
          name: '变量',
          key: 'm-project-variable',
          icon: <InteractionOutlined />
        },
        {
          name: '设置',
          isHide: !PROJECT_SET,
          key: 'm-project-setting',
          icon: <SettingOutlined />
        }
      ]
    },
    {
      subName: '组织设置',
      subKey: 'org',
      emptyMenuList: [],
      isHide: !ORG_SET,
      menuList: [
        {
          name: '项目',
          key: 'm-org-project',
          icon: <ProjectOutlined />
        },
        {
          name: '云模板',
          key: 'm-org-ct',
          icon: <LayoutOutlined />
        },
        {
          name: '变量',
          key: 'm-org-variable',
          icon: <InteractionOutlined />
        },
        {
          name: '设定',
          key: 'm-org-setting',
          icon: <FormOutlined />
        }
      ]
    },
    {
      subName: '多云管理',
      subKey: 'other',
      emptyMenuList: [],
      menuList: [
        {
          name: '总览',
          key: 'm-cloud-overview',
          icon: <DashboardOutlined />
        },
        {
          name: '云账号',
          key: 'm-cloud-account',
          icon: <CloudServerOutlined />
        },
        {
          name: '云资产',
          key: 'm-cloud-assets',
          icon: <SearchOutlined />
        },
        {
          name: '操作任务',
          key: 'm-cloud-operations',
          icon: <FormOutlined />
        },
        {
          name: '风险合规',
          key: 'm-cloud-risks',
          icon: <SafetyCertificateOutlined />
        },
        {
          name: '成本中心',
          key: 'm-cloud-costs',
          icon: <DollarOutlined />
        },
        {
          name: '事件中心',
          key: 'm-cloud-events',
          icon: <BellOutlined />
        },
        {
          name: 'ITSM工单',
          key: 'm-cloud-itsm',
          icon: <FormOutlined />
        }
      ]
    }
  ].filter(it => !it.isHide);
};

export default getMenus;
