#!/usr/bin/env python3
"""
Bedrock Claude 代理管理工具
用于管理API密钥和查看使用情况
"""

import sys
import json
import functools

import click
import requests
from tabulate import tabulate


class Config:
    """配置管理类"""
    def __init__(self):
        self.url = "http://localhost:6000"
        self.token = None


# 实例化配置
config = Config()


def login(username, password):
    """管理员登录"""
    url = f"{config.url}/login/admin"
    headers = {"Content-Type": "application/json"}
    data = {"username": username, "password": password}

    try:
        response = requests.post(url, headers=headers, json=data)
        response.raise_for_status()

        token_data = response.json()
        config.token = token_data.get("token")
        return True
    except Exception as e:
        click.echo(f"登录失败: {e}", err=True)
        return False


def check_auth(func):
    """检查是否已登录的装饰器"""

    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        if not config.token:
            username = "proxy"
            password = "hello@autel.com"

            if not login(username, password):
                sys.exit(1)

        return func(*args, **kwargs)
    return wrapper


@click.group()
def cli():
    pass


# @cli.command()
# @click.option('--username', '-u', default="hello", help='管理员用户名')
# @click.option('--password', '-p', default="123456", help='管理员密码')
# def login_admin(username, password):
#     """登录管理员账号"""
#     if login(username, password):
#         click.echo("登录成功!")


@cli.command()
@check_auth
@click.option('--name', '-n', required=True, help='API密钥名称')
def create_apikey(name):
    """创建API密钥"""
    url = f"{config.url}/admin/apikey/create"
    headers = {
        "Authorization": f"Bearer {config.token}",
        "Content-Type": "application/json"
    }
    data = {"name": name}

    try:
        response = requests.post(url, headers=headers, json=data)
        response.raise_for_status()

        apikey = response.json()
        click.echo(f"API密钥创建成功!")
        click.echo(f"名称: {apikey.get('name')}")
        click.echo(f"密钥: {apikey.get('value')}")

        # 显示警告信息
        click.echo("\n" + "-" * 60)
        click.echo("请保存此密钥，它不会再显示!")
        click.echo("-" * 60)
    except Exception as e:
        click.echo(f"创建API密钥失败: {e}", err=True)


@cli.command()
@check_auth
@click.option('--id', '-i', required=True, help='API密钥ID')
@click.confirmation_option(prompt='确定要删除此API密钥吗?')
def delete_apikey(id):
    """删除API密钥"""
    url = f"{config.url}/admin/apikey/{id}/delete"
    headers = {
        "Authorization": f"Bearer {config.token}",
        "Content-Type": "application/json"
    }

    try:
        response = requests.delete(url, headers=headers)
        response.raise_for_status()

        click.echo(f"API密钥删除成功!")
    except Exception as e:
        click.echo(f"删除API密钥失败: {e}", err=True)


@cli.command()
@check_auth
def list_apikey():
    """获取API密钥列表"""
    url = f"{config.url}/admin/apikey/list"
    headers = {
        "Authorization": f"Bearer {config.token}",
        "Content-Type": "application/json"
    }

    try:
        response = requests.get(url, headers=headers)
        response.raise_for_status()

        data = response.json()
        apikeys = data.get("api_keys", [])

        if not apikeys:
            click.echo("没有找到API密钥")
            return

        # 获取每个API密钥的使用情况
        usage_stats = {}
        for apikey in apikeys:
            # 使用新的API端点直接获取配额统计信息
            quota_url = f"{config.url}/admin/apikey/quota"
            params = {
                "name": apikey.get("name")
            }

            try:
                quota_response = requests.get(quota_url, headers=headers, params=params)
                quota_response.raise_for_status()
                quota_data = quota_response.json()

                usage_stats[apikey.get("id")] = {
                    "total_quota": quota_data.get("total_quota", 0),
                    "total_tokens": quota_data.get("total_tokens", 0),
                    "total_records": quota_data.get("total_requests", 0)
                }
            except Exception as e:
                click.echo(f"获取API密钥 {apikey.get('name')} 的使用情况失败: {e}", err=True)
                usage_stats[apikey.get("id")] = {
                    "total_quota": 0,
                    "total_tokens": 0,
                    "total_records": 0
                }

        # 准备表格数据
        table_data = []
        for apikey in apikeys:
            created_at = apikey.get("created_at", "").replace("T", " ").replace("Z", "")
            apikey_id = apikey.get("id", "")
            stats = usage_stats.get(apikey_id, {"total_quota": 0, "total_tokens": 0, "total_records": 0})

            table_data.append([
                apikey_id,
                apikey.get("name", ""),
                apikey.get("value", ""),
                created_at,
                stats["total_records"],
                stats["total_tokens"],
                stats["total_quota"],
                stats["total_quota"] * 0.002,
            ])

        # 使用tabulate打印表格
        headers = ["ID", "名称", "密钥", "创建时间", "请求次数", "总Token数", "消费总额度", "消费美元"]
        click.echo(tabulate(table_data, headers=headers, tablefmt="grid"))
    except Exception as e:
        click.echo(f"获取API密钥列表失败: {e}", err=True)


@cli.command()
@check_auth
@click.option('--page', '-p', default=1, help='页码')
@click.option('--page-size', '-s', default=20, help='每页记录数')
@click.option('--apikey', '-k', help='按API密钥名称过滤')
@click.option('--model', '-m', help='按模型名称过滤')
@click.option('--start', help='开始日期 (YYYY-MM-DD)')
@click.option('--end', help='结束日期 (YYYY-MM-DD)')
@click.option('--format', '-f', type=click.Choice(['table', 'json']), default='table', help='输出格式')
@click.option('--output', '-o', help='输出文件路径')
def list_usage(page, page_size, apikey, model, start, end, format, output):
    """获取使用记录列表"""
    url = f"{config.url}/admin/usage/list"
    headers = {
        "Authorization": f"Bearer {config.token}",
        "Content-Type": "application/json"
    }

    # 构建查询参数
    params = {
        "page": page,
        "page_size": page_size
    }

    if apikey:
        params["apikey_name"] = apikey

    if model:
        params["model_name"] = model

    if start:
        params["start_time"] = start

    if end:
        params["end_time"] = end

    try:
        response = requests.get(url, headers=headers, params=params)
        response.raise_for_status()

        data = response.json()
        total = data.get("total", 0)
        items = data.get("items", [])

        if output:
            # 保存到文件
            with open(output, 'w', encoding='utf-8') as f:
                json.dump(data, f, ensure_ascii=False, indent=2)
            click.echo(f"结果已保存到: {output}")

        if not items:
            click.echo("没有找到使用记录")
            return

        # 根据格式输出结果
        if format == 'json':
            click.echo(json.dumps(data, ensure_ascii=False, indent=2))
        else:
            # 准备表格数据
            table_data = []
            for item in items:
                created_at = item.get("created_at", "").replace("T", " ").split(".")[0]
                table_data.append([
                    item.get("id", ""),
                    item.get("apikey_name", ""),
                    item.get("model_name", ""),
                    item.get("input_tokens", 0),
                    item.get("output_tokens", 0),
                    item.get("quota", 0),
                    created_at
                ])

            # 使用tabulate打印表格
            headers = ["ID", "密钥名称", "模型", "输入令牌", "输出令牌", "消费额度", "创建时间"]
            click.echo(f"总记录数: {total} (第{page}页，每页{page_size}条)")
            click.echo(tabulate(table_data, headers=headers, tablefmt="grid"))
    except Exception as e:
        click.echo(f"获取使用记录失败: {e}", err=True)


@cli.command()
@check_auth
@click.option('--name', '-n', required=True, help='API密钥名称')
def get_apikey_quota(name):
    """获取API密钥的配额使用统计"""
    url = f"{config.url}/admin/apikey/quota"
    headers = {
        "Authorization": f"Bearer {config.token}",
        "Content-Type": "application/json"
    }
    params = {"name": name}

    try:
        response = requests.get(url, headers=headers, params=params)
        response.raise_for_status()

        data = response.json()

        # 打印统计信息
        click.echo(click.style(f"API密钥 '{name}' 的使用统计", fg="blue", bold=True))
        click.echo(f"总请求次数: {data.get('total_requests', 0)}")
        click.echo(f"总输入Token: {data.get('total_input_tokens', 0)}")
        click.echo(f"总输出Token: {data.get('total_output_tokens', 0)}")
        click.echo(f"总Token数量: {data.get('total_tokens', 0)}")
        click.echo(f"总配额消耗: {data.get('total_quota', 0)}")

    except Exception as e:
        click.echo(f"获取API密钥配额统计失败: {e}", err=True)


if __name__ == "__main__":
    cli()
