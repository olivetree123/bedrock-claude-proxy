import json
import sys

from anthropic import Anthropic
from anthropic.types import Message

client = Anthropic(base_url="http://127.0.0.1:6000", api_key="hello@autel.com")

content = """In the early 19th century, the Bennet family lives at their Longbourn estate, situated near the village of Meryton in Hertfordshire, England. Mrs. Bennet's greatest desire is to marry off her five daughters to secure their futures.

The arrival of Mr. Bingley, a rich bachelor who rents the neighbouring Netherfield estate, gives her hope that one of her daughters might contract a marriage to the advantage, because "It is a truth universally acknowledged, that a single man in possession of a good fortune, must be in want of a wife".

At a ball, the family is introduced to the Netherfield party, including Mr. Bingley, his two sisters and Mr. Darcy, his dearest friend. Mr. Bingley's friendly and cheerful manner earns him popularity among the guests. He appears interested in Jane, the eldest Bennet daughter. Mr. Darcy, reputed to be twice as wealthy as Mr. Bingley, is haughty and aloof, causing a decided dislike of him. He declines to dance with Elizabeth, the second-eldest Bennet daughter, as she is "not handsome enough". Although she jokes about it with her friend, Elizabeth is deeply offended. Despite this first impression, Mr. Darcy secretly begins to find himself drawn to Elizabeth as they continue to encounter each other at social events, appreciating her wit and frankness.

Mr. Collins, the heir to the Longbourn estate, visits the Bennet family with the intention of finding a wife among the five girls under the advice of his patroness Lady Catherine de Bourgh, also revealed to be Mr. Darcy's aunt. He decides to pursue Elizabeth. The Bennet family meets the charming army officer George Wickham, who tells Elizabeth in confidence about Mr. Darcy's unpleasant treatment of him in the past. Elizabeth, blinded by her prejudice toward Mr. Darcy, believes him."""

print("发送请求中...")

# 初始化全部回复内容
full_response = ""

# 使用stream=True参数启用流式输出
with client.messages.stream(
        model="claude-3-7-sonnet-20250219",
        max_tokens=8192,
        messages=[{
            "role":
            "user",
            "content": [
                {
                    "type": "text",
                    "text": "Analyze the document"
                },
                {
                    "type": "text",
                    "text": content,
                    "cache_control": {
                        "type": "ephemeral"
                    }
                },
            ]
        }],
) as stream:
    print("开始接收流式响应...\n")

    # 遍历流中的每个事件
    for text in stream.text_stream:
        # 打印文本块
        print(text, end="", flush=True)
        # 累积完整回复
        full_response += text

    # 获取完整的消息对象(包含usage信息)
    message = stream.get_final_message()

print("\n\n流式响应接收完毕！\n")

# 打印使用情况统计
if hasattr(message, "usage") and message.usage:
    usage = message.usage
    print("\n使用情况统计:")
    print(f"输入tokens: {usage.input_tokens}")
    print(f"输出tokens: {usage.output_tokens}")
    print(f"总tokens: {usage.input_tokens + usage.output_tokens}")
else:
    print("\n无法获取token使用情况")

# 保存完整响应到文件
with open("claude_response.txt", "w", encoding="utf-8") as f:
    f.write(full_response)
    print("\n响应已保存到 claude_response.txt")